<p align="center">
<a href="https://www.c3x.dev"><img src=".github/assets/logo.png" alt="C3X" width="120" /></a>
</p>

# C3X

Cloud cost estimation for Terraform, Terragrunt, and CloudFormation. Includes optimization recommendations, budget guardrails, what-if analysis, and fully offline mode. No API key required.

<p>
<a href="https://www.c3x.dev/docs/"><img alt="Docs" src="https://img.shields.io/badge/docs-get%20started-brightgreen"/></a>
<a href="https://github.com/c3xdev/c3x/releases"><img alt="Release" src="https://img.shields.io/github/v/release/c3xdev/c3x"/></a>
<a href="https://github.com/c3xdev/c3x/blob/main/LICENSE"><img alt="License" src="https://img.shields.io/github/license/c3xdev/c3x"/></a>
</p>

## Get started

```sh
brew install c3xdev/tap/c3x
```

```sh
c3x estimate --path /path/to/terraform
```

Or use Docker:

```sh
docker pull c3xdev/c3x
docker run --rm -v $(pwd):/workspace c3xdev/c3x estimate --path /workspace
```

See the [quick start guide](https://www.c3x.dev/docs/quickstart) for more details.

## Cost estimates

```
$ c3x estimate --path examples/terraform --usage-file examples/terraform/c3x-usage.yml

Project: main

 Name                                                        Monthly Qty  Unit              Monthly Cost

 aws_db_instance.postgres
 ├─ Database instance (on-demand, Multi-AZ, db.r5.large)             730  hours                  $365.00
 └─ Storage (general purpose SSD, gp3)                               200  GB                      $46.00

 aws_instance.api
 ├─ Instance usage (Linux/UNIX, on-demand, m5.xlarge)                730  hours                  $140.16
 ├─ root_block_device
 │  └─ Storage (general purpose SSD, gp2)                            100  GB                      $10.00
 └─ ebs_block_device[0]
    └─ Storage (general purpose SSD, gp3)                            500  GB                      $40.00

 aws_nat_gateway.main
 ├─ NAT gateway                                                      730  hours                   $32.85
 └─ Data processed                                        Monthly cost depends on usage: $0.045 per GB

 aws_lb.api
 ├─ Application load balancer                                        730  hours                   $16.43
 └─ Load balancer capacity units                          Monthly cost depends on usage: $5.84 per LCU

 OVERALL TOTAL                                                                                   $650.44
```

## Cost diffs

```
$ c3x diff --path . --compare-to main

Project: main

 ~ aws_instance.api
   ├─ Instance usage (Linux/UNIX, on-demand, m5.xlarge -> m6i.2xlarge)    +$170.24 ($140.16 -> $310.40)
   └─ ebs_block_device[0]
      └─ Storage (general purpose SSD, gp3, 500 -> 1000 GB)               +$40.00 ($40.00 -> $80.00)

 Monthly cost change: +$210.24 ($650.44 -> $860.68)
```

## Optimization recommendations

```sh
c3x recommend --path .
```

```
3 recommendation(s) found:

  1. Upgrade to newer instance generation (m5.xlarge -> m7i.xlarge)
     Resource: aws_instance.web (aws_instance)
     The m7i family has better price-performance than m5.

  2. Switch EBS volume from gp2 to gp3
     Resource: aws_ebs_volume.data (aws_ebs_volume)
     gp3 volumes are up to 20% cheaper with better baseline performance.

  3. Consider VPC endpoints to reduce NAT Gateway costs
     Resource: aws_nat_gateway.main (aws_nat_gateway)
     NAT Gateway charges $0.045/GB. VPC endpoints can eliminate these charges.
```

## Budget guardrails

Enforce cost limits in CI/CD pipelines. No paid subscription required.

```sh
c3x estimate --path . --budget 1000
c3x diff --path . --compare-to baseline.json --budget-increase 20
```

## What-if analysis

Test cost impact of changes without modifying Terraform code.

```sh
c3x estimate --path . --what-if 'aws_instance.web.instance_type=m6i.8xlarge'
c3x estimate --path . --what-if 'aws_db_instance.main.multi_az=true'
```

## PR comments

Post cost estimates in pull requests. Two steps, no secrets.

```yaml
- uses: actions/checkout@v4
- uses: c3xdev/setup-c3x@v1
  with:
    path: .
```

Install the [C3X Cloud](https://github.com/apps/c3x-cloud) app for branded comments with the C3X logo. Also supports GitLab, Bitbucket, Azure Repos, Atlantis, and Spacelift.

## Offline mode

Download pricing data once. Estimate without network calls.

```sh
c3x pricing sync --providers aws,azure
c3x estimate --path . --offline
```

## Self-hosted pricing API

Run your own [pricing API](https://github.com/c3xdev/c3x-pricing-api) that scrapes directly from AWS, Azure, and GCP.

```sh
export C3X_SELF_HOSTED=true
export C3X_PRICING_API_ENDPOINT=http://localhost:4000
c3x estimate --path .
```

## Supported resources

Over 1,100 Terraform resources across [AWS](https://www.c3x.dev/docs/supported-resources), [Azure](https://www.c3x.dev/docs/supported-resources), and [Google Cloud](https://www.c3x.dev/docs/supported-resources). Usage-based resources (Lambda, S3, data transfer) are supported via [usage files](https://www.c3x.dev/docs/usage-file).

## Roadmap

- Compare estimates to real cloud bills (AWS Cost Explorer, Azure Cost Management)
- Pulumi, AWS CDK, Azure Bicep support
- Historical cost tracking across commits and PRs
- VS Code and JetBrains inline cost estimates

## Contributing

Open a thread in [GitHub Discussions](https://github.com/c3xdev/c3x/discussions) before submitting a PR. See the [contribution guide](CONTRIBUTING.md) for details.

## License

[Apache License 2.0](https://choosealicense.com/licenses/apache-2.0/)
