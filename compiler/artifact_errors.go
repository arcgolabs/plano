package compiler

import "errors"

var (
	errNilArtifactReceiver  = errors.New("unmarshal artifact json: nil receiver")
	errMissingListElem      = errors.New("artifact type list: missing elem")
	errMissingMapElem       = errors.New("artifact type map: missing elem")
	errMissingRefData       = errors.New("artifact value ref: missing data")
	errMissingDurationData  = errors.New("artifact value duration: missing data")
	errMissingSizeData      = errors.New("artifact value size: missing data")
	errNilArtifactListCodec = errors.New("artifact list codec is nil")
	errNilArtifactMapCodec  = errors.New("artifact map codec is nil")
)

func errWrapArtifactJSON(message string, err error) error {
	return wrapCompilerErrorf(err, "%s", message)
}

func validateArtifactSchemaVersion(version string) error {
	switch version {
	case "", "plano.artifact/v1", ArtifactSchemaVersion:
		return nil
	default:
		return compilerErrorf("artifact schema version %q is not supported", version)
	}
}
