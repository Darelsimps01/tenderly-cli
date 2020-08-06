package providers

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/tenderly/tenderly-cli/model"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"strings"
)

func GetContracts(
	buildDir string,
	networkIDs []string,
	objects ...*model.StateObject,
) ([]Contract, int, error) {
	files, err := ioutil.ReadDir(buildDir)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed listing build files")
	}

	networkIDFilterMap := make(map[string]bool)
	for _, networkID := range networkIDs {
		networkIDFilterMap[networkID] = true
	}
	objectMap := make(map[string]*model.StateObject)
	for _, object := range objects {
		if object.Code == nil || len(object.Code) == 0 {
			continue
		}
		objectMap[hexutil.Encode(object.Code)] = object
	}

	hasNetworkFilters := len(networkIDFilterMap) > 0

	sources := make(map[string]bool)
	var contracts []Contract
	var numberOfContractsWithANetwork int
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(buildDir, file.Name())
		data, err := ioutil.ReadFile(filePath)

		if err != nil {
			return nil, 0, errors.Wrap(err, "failed reading build file")
		}

		var contract Contract
		err = json.Unmarshal(data, &contract)
		if err != nil {
			return nil, 0, errors.Wrap(err, "failed parsing build file")
		}

		if contract.Networks == nil {
			contract.Networks = make(map[string]ContractNetwork)
		}

		sources[contract.SourcePath] = true
		for _, node := range contract.Ast.Nodes {
			if node.NodeType != "ImportDirective" {
				continue
			}

			absPath := node.AbsolutePath
			if runtime.GOOS == "windows" && strings.HasPrefix(absPath, "/") {
				absPath = strings.ReplaceAll(absPath, "/", "\\")
				absPath = strings.TrimPrefix(absPath, "\\")
				absPath = strings.Replace(absPath, "\\", ":\\", 1)
			}

			if !sources[absPath] {
				sources[absPath] = false
			}
		}

		if object := objectMap[contract.DeployedBytecode]; object != nil && len(networkIDs) == 1 {
			if _, ok := contract.Networks[networkIDs[0]]; !ok {
				contract.Networks[networkIDs[0]] = ContractNetwork{
					Links:   nil, // @TODO: Libraries
					Address: object.Address,
				}
			}
		}

		if hasNetworkFilters {
			for networkID := range contract.Networks {
				if !networkIDFilterMap[networkID] {
					delete(contract.Networks, networkID)
				}
			}
		}

		contracts = append(contracts, contract)
		numberOfContractsWithANetwork += len(contract.Networks)
	}

	for path, included := range sources {
		if !included {
			source, err := ioutil.ReadFile(path)
			if err != nil {
				return nil, 0, errors.Wrap(err, "failed reading contract source file")
			}

			contracts = append(contracts, Contract{
				Source:     string(source),
				SourcePath: path,
			})
		}
	}

	return contracts, numberOfContractsWithANetwork, nil
}