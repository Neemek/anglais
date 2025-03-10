#!/bin/zsh

echo '== Building CLI =='
cd cli
go build .
cd ..


echo '== Testing anglais =='

for file in ./tests/*.ang; do
  echo "-- Test-running file $file --"

  if ! ./cli/cli run "$file"; then
    echo "-- Error --"
    exit 1
  else
    echo "-- Success --"
  fi
done

echo '== Successfully ran all tests =='
