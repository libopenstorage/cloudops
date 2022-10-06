gen:
	go run generator.go
	sed -i '1 s/^/# This is an auto-generated file. DO NOT EDIT.\n/' oracle.yaml
	cp oracle.yaml ../testspecs/
	mv oracle.yaml ../../../specs/decisionmatrix/