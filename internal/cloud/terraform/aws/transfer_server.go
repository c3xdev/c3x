package aws

import (
	"sort"

	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getTransferServerRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_transfer_server",
		CoreRFunc: newTransferServer,
	}
}

func newTransferServer(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Get("region").String()
	protocols := []string{}

	if d.Get("protocols").Exists() {
		for _, data := range d.Get("protocols").Array() {
			protocols = append(protocols, data.String())
		}

		sort.Strings(protocols)
	} else {
		defaultProtocol := "SFTP"
		protocols = append(protocols, defaultProtocol)
	}

	t := &aws.TransferServer{
		Address:   d.Address,
		Region:    region,
		Protocols: protocols,
	}

	return t
}
