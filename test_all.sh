#!/bin/bash

echo '=== Building CLI ==='
cd cli || exit 1
if ! go build .; then
  echo "=== Had error building CLI ==="
  exit 1
else
  echo "=+= Successfully built CLI =+="
fi
cd ..

echo "=== Running go core tests ==="
cd core || exit 1
if ! go test .; then
  echo "=x= Core testing failed =x= "
  exit 1
else
  echo "=+= Successfully ran core tests =+="
fi
cd ..

errors=()

echo '=== Testing anglais ==='

# read files
for file in $(find tests -type f); do
  echo "-v- Test-running file $file -v-"

  if ! ./cli/cli run "$file"; then
    echo "-x- Error -x-"
    errors+=("$file")
  else
    echo "-+- Success -+-"
  fi
done


if [ 0 -ne "$(wc -w <<< "${errors[@]}")" ]; then
  echo "== Errors occured while executing =="
  echo "erroring files: $(printf '%s ' "${errors[@]}")"
  exit 1
else
  echo '== Successfully ran all tests =='
fi
