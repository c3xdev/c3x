package terraform

import (
	"bytes"
	"compress/gzip"
	"crypto/md5"  //nolint:gosec // Terraform's md5() is a checksum, not security.
	"crypto/sha1" //nolint:gosec // Terraform's sha1() likewise.
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"math/big"
	"net/netip"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	ctyjson "github.com/zclconf/go-cty/cty/json"
	"sigs.k8s.io/yaml"
)

// Terraform and OpenTofu built-ins that go-cty's stdlib does not provide.
//
// A function the evaluator doesn't know is never the user's intent, and
// the failure is silent where it hurts: an unknown function in an
// attribute leaves it unset, so the catalog prices its default instead,
// and one in a local takes every resource that depends on it out of the
// estimate. `for_each = { for ... : az => cidrsubnet(...) }` dropped three
// NAT gateways that way. These are implemented to the published
// Terraform/OpenTofu semantics, with the documented examples as tests.
//
// The filesystem functions live in file_functions.go, because they
// depend on the directory an expression is evaluated from. Not provided:
// the non-deterministic ones (timestamp, uuid, bcrypt) and pathexpand,
// which would put the host's home directory into an estimate. Calls to
// any function still missing are reported by warnUnknownFunctions.
func extraFunctions() map[string]function.Function {
	return map[string]function.Function{
		// Collections.
		"one": oneFunc,
		"sum": sumFunc,
		"alltrue": boolListFunc(func(bs []bool) bool {
			for _, b := range bs {
				if !b {
					return false
				}
			}
			return true
		}),
		"anytrue": boolListFunc(func(bs []bool) bool {
			for _, b := range bs {
				if b {
					return true
				}
			}
			return false
		}),

		// Strings.
		"startswith":     stringPredicate(strings.HasPrefix),
		"endswith":       stringPredicate(strings.HasSuffix),
		"strcontains":    stringPredicate(strings.Contains),
		"templatestring": templateStringFunc(),
		// Pure path string manipulation; no filesystem access, so not gated
		// with the file functions.
		"basename": stringToString(func(s string) (string, error) { return filepath.Base(s), nil }),
		"dirname":  stringToString(func(s string) (string, error) { return filepath.Dir(s), nil }),

		// Sensitivity markers have no meaning for a static estimate.
		"sensitive":       identityFunc,
		"nonsensitive":    identityFunc,
		"ephemeralasnull": identityFunc,
		"issensitive": function.New(&function.Spec{
			Params: []function.Parameter{{Name: "value", Type: cty.DynamicPseudoType, AllowNull: true, AllowUnknown: true}},
			Type:   function.StaticReturnType(cty.Bool),
			Impl: func([]cty.Value, cty.Type) (cty.Value, error) {
				return cty.False, nil
			},
		}),

		// Encoding.
		"base64encode": stringToString(func(s string) (string, error) {
			return base64.StdEncoding.EncodeToString([]byte(s)), nil
		}),
		"base64decode": stringToString(func(s string) (string, error) {
			b, err := base64.StdEncoding.DecodeString(s)
			return string(b), err
		}),
		"base64gzip": stringToString(func(s string) (string, error) {
			var buf bytes.Buffer
			w := gzip.NewWriter(&buf)
			if _, err := w.Write([]byte(s)); err != nil {
				return "", err
			}
			if err := w.Close(); err != nil {
				return "", err
			}
			return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
		}),
		"base64gunzip": stringToString(func(s string) (string, error) { // OpenTofu
			raw, err := base64.StdEncoding.DecodeString(s)
			if err != nil {
				return "", err
			}
			r, err := gzip.NewReader(bytes.NewReader(raw))
			if err != nil {
				return "", err
			}
			out, err := io.ReadAll(r)
			return string(out), err
		}),
		"urlencode":  stringToString(func(s string) (string, error) { return url.QueryEscape(s), nil }),
		"urldecode":  stringToString(url.QueryUnescape), // OpenTofu
		"yamldecode": yamlDecodeFunc,
		"yamlencode": yamlEncodeFunc,

		// Hashes.
		"md5":          hashFunc(md5.New, hex.EncodeToString),
		"sha1":         hashFunc(sha1.New, hex.EncodeToString),
		"sha256":       hashFunc(sha256.New, hex.EncodeToString),
		"sha512":       hashFunc(sha512.New, hex.EncodeToString),
		"base64sha256": hashFunc(sha256.New, base64.StdEncoding.EncodeToString),
		"base64sha512": hashFunc(sha512.New, base64.StdEncoding.EncodeToString),

		// Networking.
		"cidrhost":     cidrHostFunc,
		"cidrnetmask":  cidrNetmaskFunc,
		"cidrsubnet":   cidrSubnetFunc,
		"cidrsubnets":  cidrSubnetsFunc,
		"cidrcontains": cidrContainsFunc, // OpenTofu
	}
}

var identityFunc = function.New(&function.Spec{
	Params: []function.Parameter{{Name: "value", Type: cty.DynamicPseudoType, AllowNull: true, AllowUnknown: true, AllowDynamicType: true}},
	Type:   func(args []cty.Value) (cty.Type, error) { return args[0].Type(), nil },
	Impl:   func(args []cty.Value, _ cty.Type) (cty.Value, error) { return args[0], nil },
})

// one returns the only element of a list or set, null for an empty one,
// and an error for more than one.
var oneFunc = function.New(&function.Spec{
	Params: []function.Parameter{{Name: "list", Type: cty.DynamicPseudoType}},
	Type: func(args []cty.Value) (cty.Type, error) {
		ty := args[0].Type()
		switch {
		case ty.IsListType() || ty.IsSetType():
			return ty.ElementType(), nil
		case ty.IsTupleType():
			if n := len(ty.TupleElementTypes()); n == 1 {
				return ty.TupleElementTypes()[0], nil
			} else if n == 0 {
				return cty.DynamicPseudoType, nil
			}
			return cty.NilType, fmt.Errorf("must be a list, set, or tuple value with either zero or one elements")
		}
		return cty.NilType, fmt.Errorf("must be a list, set, or tuple value with either zero or one elements")
	},
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		v := args[0]
		if !v.IsKnown() {
			return cty.UnknownVal(retType), nil
		}
		switch v.LengthInt() {
		case 0:
			return cty.NullVal(retType), nil
		case 1:
			it := v.ElementIterator()
			it.Next()
			_, e := it.Element()
			return e, nil
		}
		return cty.NilVal, fmt.Errorf("must be a list, set, or tuple value with either zero or one elements")
	},
})

var sumFunc = function.New(&function.Spec{
	Params: []function.Parameter{{Name: "list", Type: cty.DynamicPseudoType}},
	Type:   function.StaticReturnType(cty.Number),
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		v := args[0]
		if !v.CanIterateElements() || v.LengthInt() == 0 {
			return cty.NilVal, fmt.Errorf("cannot sum an empty list")
		}
		total := new(big.Float)
		for it := v.ElementIterator(); it.Next(); {
			_, e := it.Element()
			if e.IsNull() || !e.IsKnown() || e.Type() != cty.Number {
				return cty.UnknownVal(cty.Number), nil
			}
			total.Add(total, e.AsBigFloat())
		}
		return cty.NumberVal(total), nil
	},
})

func boolListFunc(fn func([]bool) bool) function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{{Name: "list", Type: cty.List(cty.Bool)}},
		Type:   function.StaticReturnType(cty.Bool),
		Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
			var bs []bool
			for it := args[0].ElementIterator(); it.Next(); {
				_, e := it.Element()
				if !e.IsKnown() || e.IsNull() {
					return cty.UnknownVal(cty.Bool), nil
				}
				bs = append(bs, e.True())
			}
			return cty.BoolVal(fn(bs)), nil
		},
	})
}

func stringPredicate(fn func(s, sub string) bool) function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{{Name: "str", Type: cty.String}, {Name: "substr", Type: cty.String}},
		Type:   function.StaticReturnType(cty.Bool),
		Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
			return cty.BoolVal(fn(args[0].AsString(), args[1].AsString())), nil
		},
	})
}

func stringToString(fn func(string) (string, error)) function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{{Name: "str", Type: cty.String}},
		Type:   function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
			out, err := fn(args[0].AsString())
			if err != nil {
				return cty.NilVal, err
			}
			return cty.StringVal(out), nil
		},
	})
}

func hashFunc(newHash func() hash.Hash, encode func([]byte) string) function.Function {
	return stringToString(func(s string) (string, error) {
		h := newHash()
		h.Write([]byte(s))
		return encode(h.Sum(nil)), nil
	})
}

// templateStringFunc renders a string as a template against a map of
// variables, as templatestring(tpl, { name = value }) does. It is built
// on demand rather than held in a package variable because its
// implementation needs the full function table, which includes itself.
func templateStringFunc() function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{
			{Name: "template", Type: cty.String},
			{Name: "vars", Type: cty.DynamicPseudoType},
		},
		Type: function.StaticReturnType(cty.DynamicPseudoType),
		Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
			expr, diags := hclsyntax.ParseTemplate([]byte(args[0].AsString()), "templatestring", hcl.InitialPos)
			if diags.HasErrors() {
				return cty.NilVal, diags
			}
			vars := map[string]cty.Value{}
			if v := args[1]; v.CanIterateElements() {
				for it := v.ElementIterator(); it.Next(); {
					k, e := it.Element()
					vars[k.AsString()] = e
				}
			}
			val, diags := expr.Value(&hcl.EvalContext{Variables: vars, Functions: terraformFunctions()})
			if diags.HasErrors() {
				return cty.NilVal, diags
			}
			return val, nil
		},
	})
}

var yamlDecodeFunc = function.New(&function.Spec{
	Params: []function.Parameter{{Name: "src", Type: cty.String}},
	Type: func(args []cty.Value) (cty.Type, error) {
		if !args[0].IsKnown() {
			return cty.DynamicPseudoType, nil
		}
		js, err := yaml.YAMLToJSON([]byte(args[0].AsString()))
		if err != nil {
			return cty.NilType, err
		}
		return ctyjson.ImpliedType(js)
	},
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		js, err := yaml.YAMLToJSON([]byte(args[0].AsString()))
		if err != nil {
			return cty.NilVal, err
		}
		return ctyjson.Unmarshal(js, retType)
	},
})

var yamlEncodeFunc = function.New(&function.Spec{
	Params: []function.Parameter{{Name: "value", Type: cty.DynamicPseudoType, AllowNull: true}},
	Type:   function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		js, err := ctyjson.Marshal(args[0], args[0].Type())
		if err != nil {
			return cty.NilVal, err
		}
		y, err := yaml.JSONToYAML(js)
		if err != nil {
			return cty.NilVal, err
		}
		return cty.StringVal(string(y)), nil
	},
})

// --- CIDR arithmetic, to Terraform's semantics ---

func parsePrefix(s string) (netip.Prefix, error) {
	p, err := netip.ParsePrefix(s)
	if err != nil {
		return netip.Prefix{}, fmt.Errorf("invalid CIDR expression: %w", err)
	}
	return p.Masked(), nil
}

func addrToInt(a netip.Addr) *big.Int { return new(big.Int).SetBytes(a.AsSlice()) }

func intToAddr(n *big.Int, is4 bool) (netip.Addr, error) {
	size := 16
	if is4 {
		size = 4
	}
	b := n.Bytes()
	if len(b) > size {
		return netip.Addr{}, fmt.Errorf("address out of range")
	}
	buf := make([]byte, size)
	copy(buf[size-len(b):], b)
	a, _ := netip.AddrFromSlice(buf)
	return a, nil
}

func bitsFor(p netip.Prefix) int { return p.Addr().BitLen() }

// subnetOf extends p by newbits and selects subnet number num.
func subnetOf(p netip.Prefix, newbits int, num *big.Int) (netip.Prefix, error) {
	newLen := p.Bits() + newbits
	if newbits < 0 || newLen > bitsFor(p) {
		return netip.Prefix{}, fmt.Errorf("insufficient address space to extend prefix of %d by %d", p.Bits(), newbits)
	}
	limit := new(big.Int).Lsh(big.NewInt(1), uint(newbits))
	if num.Sign() < 0 || num.Cmp(limit) >= 0 {
		return netip.Prefix{}, fmt.Errorf("prefix extension of %d does not accommodate a subnet numbered %s", newbits, num)
	}
	base := addrToInt(p.Addr())
	shift := uint(bitsFor(p) - newLen)
	base.Or(base, new(big.Int).Lsh(num, shift))
	a, err := intToAddr(base, p.Addr().Is4())
	if err != nil {
		return netip.Prefix{}, err
	}
	return netip.PrefixFrom(a, newLen), nil
}

var cidrSubnetFunc = function.New(&function.Spec{
	Params: []function.Parameter{{Name: "prefix", Type: cty.String}, {Name: "newbits", Type: cty.Number}, {Name: "netnum", Type: cty.Number}},
	Type:   function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		p, err := parsePrefix(args[0].AsString())
		if err != nil {
			return cty.NilVal, err
		}
		nb, _ := args[1].AsBigFloat().Int64()
		num, _ := args[2].AsBigFloat().Int(nil)
		s, err := subnetOf(p, int(nb), num)
		if err != nil {
			return cty.NilVal, err
		}
		return cty.StringVal(s.String()), nil
	},
})

var cidrSubnetsFunc = function.New(&function.Spec{
	Params:   []function.Parameter{{Name: "prefix", Type: cty.String}},
	VarParam: &function.Parameter{Name: "newbits", Type: cty.Number},
	Type:     function.StaticReturnType(cty.List(cty.String)),
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		p, err := parsePrefix(args[0].AsString())
		if err != nil {
			return cty.NilVal, err
		}
		if len(args) == 1 {
			return cty.ListValEmpty(cty.String), nil
		}
		// Allocate consecutively, aligning each subnet to its own size.
		width := bitsFor(p)
		next := addrToInt(p.Addr())
		end := new(big.Int).Add(addrToInt(p.Addr()), new(big.Int).Lsh(big.NewInt(1), uint(width-p.Bits())))
		out := make([]cty.Value, 0, len(args)-1)
		for _, a := range args[1:] {
			nb, _ := a.AsBigFloat().Int64()
			newLen := p.Bits() + int(nb)
			if nb < 1 || newLen > width {
				return cty.NilVal, fmt.Errorf("invalid newbits %d for prefix %s", nb, p)
			}
			size := new(big.Int).Lsh(big.NewInt(1), uint(width-newLen))
			if rem := new(big.Int).Mod(next, size); rem.Sign() != 0 {
				next.Add(next, new(big.Int).Sub(size, rem))
			}
			if new(big.Int).Add(next, size).Cmp(end) > 0 {
				return cty.NilVal, fmt.Errorf("not enough remaining address space for a subnet with a prefix of %d bits", newLen)
			}
			addr, err := intToAddr(next, p.Addr().Is4())
			if err != nil {
				return cty.NilVal, err
			}
			out = append(out, cty.StringVal(netip.PrefixFrom(addr, newLen).String()))
			next = new(big.Int).Add(next, size)
		}
		return cty.ListVal(out), nil
	},
})

var cidrHostFunc = function.New(&function.Spec{
	Params: []function.Parameter{{Name: "prefix", Type: cty.String}, {Name: "hostnum", Type: cty.Number}},
	Type:   function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		p, err := parsePrefix(args[0].AsString())
		if err != nil {
			return cty.NilVal, err
		}
		host, _ := args[1].AsBigFloat().Int(nil)
		hostBits := uint(bitsFor(p) - p.Bits())
		size := new(big.Int).Lsh(big.NewInt(1), hostBits)
		if host.Sign() < 0 { // negative counts back from the end of the range
			host.Add(host, size)
		}
		if host.Sign() < 0 || host.Cmp(size) >= 0 {
			return cty.NilVal, fmt.Errorf("prefix of %d does not accommodate a host numbered %s", p.Bits(), args[1].AsBigFloat().String())
		}
		n := addrToInt(p.Addr())
		n.Or(n, host)
		a, err := intToAddr(n, p.Addr().Is4())
		if err != nil {
			return cty.NilVal, err
		}
		return cty.StringVal(a.String()), nil
	},
})

var cidrNetmaskFunc = function.New(&function.Spec{
	Params: []function.Parameter{{Name: "prefix", Type: cty.String}},
	Type:   function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		p, err := parsePrefix(args[0].AsString())
		if err != nil {
			return cty.NilVal, err
		}
		if !p.Addr().Is4() {
			return cty.NilVal, fmt.Errorf("IPv6 addresses cannot have a netmask: %s", p)
		}
		mask := uint32(0xffffffff) << (32 - uint(p.Bits()))
		if p.Bits() == 0 {
			mask = 0
		}
		a := netip.AddrFrom4([4]byte{byte(mask >> 24), byte(mask >> 16), byte(mask >> 8), byte(mask)})
		return cty.StringVal(a.String()), nil
	},
})

// cidrcontains is an OpenTofu addition: does the prefix contain an
// address, or wholly contain another prefix.
var cidrContainsFunc = function.New(&function.Spec{
	Params: []function.Parameter{{Name: "containing_prefix", Type: cty.String}, {Name: "contained_ip_or_prefix", Type: cty.String}},
	Type:   function.StaticReturnType(cty.Bool),
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		outer, err := parsePrefix(args[0].AsString())
		if err != nil {
			return cty.NilVal, err
		}
		s := args[1].AsString()
		if strings.Contains(s, "/") {
			inner, err := parsePrefix(s)
			if err != nil {
				return cty.NilVal, err
			}
			return cty.BoolVal(inner.Bits() >= outer.Bits() && outer.Contains(inner.Addr())), nil
		}
		a, err := netip.ParseAddr(s)
		if err != nil {
			return cty.NilVal, fmt.Errorf("invalid IP address: %w", err)
		}
		return cty.BoolVal(outer.Contains(a)), nil
	},
})
