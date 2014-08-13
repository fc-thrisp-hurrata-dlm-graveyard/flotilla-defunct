package fleet

type (
	EngineAsset struct {
		AssetFunc     func(string) ([]byte, error)
		AssetListFunc func() []string
		AssetDirFunc  func(string) ([]string, error)
	}

	IEngineAsset interface {
		Asset(string) ([]byte, error)
	}
)

func (e *EngineAsset) Asset(name string) ([]byte, error) {
	res, err := e.AssetFunc(name)
	return res, err
}

func (e *EngineAsset) AssetList() []string {
	res := e.AssetListFunc()
	return res
}

func (e *EngineAsset) AssetDir(name string) ([]string, error) {
	res, err := e.AssetDirFunc(name)
	return res, err
}
