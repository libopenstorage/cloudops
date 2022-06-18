gen:
	go run generator.go
	sed -i '1 s/^/# This is an auto-generated file. DO NOT EDIT.\n/' ibm.yaml
	mv ibm.yaml ../../../specs/decisionmatrix/
