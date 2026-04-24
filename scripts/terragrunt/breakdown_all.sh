#!/usr/bin/env bash

# See https://www.c3x.dev/docs/iac_tools/terragrunt for usage docs

# Output terraform plans
terragrunt run-all plan -out=c3x-plan

# Loop through plans and output c3x JSONs
planfiles=($(find . -name "c3x-plan" | tr '\n' ' '))
for planfile in "${planfiles[@]}"; do
  echo "Running terraform show for $planfile";
  dir=$(dirname $planfile)
  cd $dir
  terraform show -json $(basename $planfile) > c3x-plan.json
  cd -
  c3x breakdown --path $dir/c3x-plan.json --format json > $dir/c3x-out.json
  rm $planfile
done

# Run c3x output to merge the results
jsonfiles=($(find . -name "c3x-out.json" | tr '\n' ' '))
c3x output --format html $(echo ${jsonfiles[@]/#/--path }) > c3x-report.html
c3x output --format table $(echo ${jsonfiles[@]/#/--path })
echo "Also saved HTML report in c3x-report.html"

# Tidy up
rm $jsonfiles
