#!/bin/sh

inputz=./sample.d/input.zip
output=./sample.d/output.asn1.der.dat

geninput(){
	echo generating input zip file...

	mkdir -p sample.d

	printf hw1 > ./sample.d/hw1.txt
	printf hw2 > ./sample.d/hw2.txt
	printf hwIII > ./sample.d/hw3.txt

	ls ./sample.d/*.txt |
		zip \
			-0 \
			-@ \
			-T \
			-v \
			-o \
			"${inputz}"
}

test -f "${inputz}" || geninput

unzip -lv "${inputz}"

export ENV_ZIP_FILENAME="${inputz}"

echo
echo creating basic zip info with offset...
./zip2basicinfo |
	dd \
		if=/dev/stdin \
		of="${output}" \
		bs=1048576 \
		status=none

which jq    | fgrep -q jq    || exec sh -c 'echo jq missing.; exit 1'
which dasel | fgrep -q dasel || exec sh -c 'echo dasel missing.; exit 1'
which bat   | fgrep -q bat   || exec sh -c 'echo bat missing.; exit 1'

echo converting the basic info to json...
cat "${output}" |
	xxd -ps |
	tr -d '\n' |
	python3 \
		-m asn1tools \
		convert \
		-i der \
		-o jer \
		./zipinfo.asn \
		BasicZipIndexInfo \
		- |
	jq '.[]' |
	dasel \
		--read=json \
		--write=yaml |
	bat --language=yaml

cat "${inputz}" |
	tail --bytes=+75 |
	head --bytes=3 |
	xxd

cat "${inputz}" |
	tail --bytes=+152 |
	head --bytes=3 |
	xxd

cat "${inputz}" |
	tail --bytes=+229 |
	head --bytes=5 |
	xxd
