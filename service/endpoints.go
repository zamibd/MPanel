package service

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/zamibd/MPanel/database"
	"github.com/zamibd/MPanel/database/model"
	"github.com/zamibd/MPanel/util/common"

	"gorm.io/gorm"
)

type EndpointService struct {
	WarpService
}

func (o *EndpointService) GetAll() (*[]map[string]interface{}, error) {
	db := database.GetDB()
	endpoints := []*model.Endpoint{}
	err := db.Model(model.Endpoint{}).Scan(&endpoints).Error
	if err != nil {
		return nil, err
	}
	var data []map[string]interface{}
	for _, endpoint := range endpoints {
		epData := map[string]interface{}{
			"id":   endpoint.Id,
			"type": endpoint.Type,
			"tag":  endpoint.Tag,
			"ext":  endpoint.Ext,
		}
		if endpoint.Options != nil {
			var restFields map[string]json.RawMessage
			if err := json.Unmarshal(endpoint.Options, &restFields); err != nil {
				return nil, err
			}
			for k, v := range restFields {
				epData[k] = v
			}
		}
		data = append(data, epData)
	}
	return &data, nil
}

func (o *EndpointService) GetAllConfig(db *gorm.DB) ([]json.RawMessage, error) {
	var endpointsJson []json.RawMessage
	var endpoints []*model.Endpoint
	err := db.Model(model.Endpoint{}).Scan(&endpoints).Error
	if err != nil {
		return nil, err
	}
	for _, endpoint := range endpoints {
		endpointJson, err := endpoint.MarshalJSON()
		if err != nil {
			return nil, err
		}
		endpointsJson = append(endpointsJson, endpointJson)
	}
	return endpointsJson, nil
}

func (s *EndpointService) Save(tx *gorm.DB, act string, data json.RawMessage) error {
	var err error

	switch act {
	case "new", "edit":
		var endpoint model.Endpoint
		err = endpoint.UnmarshalJSON(data)
		if err != nil {
			return err
		}

		if endpoint.Type == "wireguard" || endpoint.Type == "warp" {
			var opts map[string]interface{}
			if len(endpoint.Options) > 0 {
				json.Unmarshal(endpoint.Options, &opts)
			} else {
				opts = make(map[string]interface{})
			}

			// Auto-generate endpoint private key if missing
			privKey, _ := opts["private_key"].(string)
			if privKey == "" || privKey == "auto" {
				var ss ServerService
				keys := ss.GenKeypair("wireguard", "")
				if len(keys) == 2 {
					opts["private_key"] = strings.TrimPrefix(keys[0], "PrivateKey: ")
					opts["public_key"] = strings.TrimPrefix(keys[1], "PublicKey: ") // Saved for reference
				}
			}

			// Auto-generate keys for peers if public_key is 'auto' or missing
			if peersList, ok := opts["peers"].([]interface{}); ok {
				for _, p := range peersList {
					if peer, ok := p.(map[string]interface{}); ok {
						pubKey, _ := peer["public_key"].(string)
						if pubKey == "" || pubKey == "auto" {
							var ss ServerService
							keys := ss.GenKeypair("wireguard", "")
							if len(keys) == 2 {
								peer["public_key"] = strings.TrimPrefix(keys[1], "PublicKey: ")
								peer["private_key"] = strings.TrimPrefix(keys[0], "PrivateKey: ") // Saved for user reference
							}
						}
					}
				}
			}
			endpoint.Options, _ = json.Marshal(opts)

			if act == "new" && endpoint.Type == "warp" {
				err = s.WarpService.RegisterWarp(&endpoint)
				if err != nil {
					return err
				}
			} else if act == "edit" && endpoint.Type == "warp" {
				var old_license string
				err = tx.Model(model.Endpoint{}).Select("ext->>'license_key'").Where("id = ?", endpoint.Id).Find(&old_license).Error
				if err != nil {
					return err
				}
				err = s.WarpService.SetWarpLicense(old_license, &endpoint)
				if err != nil {
					return err
				}
			}
		}

		if corePtr.IsRunning() {
			configData, err := endpoint.MarshalJSON()
			if err != nil {
				return err
			}
			if act == "edit" {
				var oldTag string
				err = tx.Model(model.Endpoint{}).Select("tag").Where("id = ?", endpoint.Id).Find(&oldTag).Error
				if err != nil {
					return err
				}
				err = corePtr.RemoveEndpoint(oldTag)
				if err != nil && err != os.ErrInvalid {
					return err
				}
			}
			err = corePtr.AddEndpoint(configData)
			if err != nil {
				return err
			}
		}

		err = tx.Save(&endpoint).Error
		if err != nil {
			return err
		}
	case "del":
		var tag string
		err = json.Unmarshal(data, &tag)
		if err != nil {
			return err
		}
		if corePtr.IsRunning() {
			err = corePtr.RemoveEndpoint(tag)
			if err != nil && err != os.ErrInvalid {
				return err
			}
		}
		err = tx.Where("tag = ?", tag).Delete(model.Endpoint{}).Error
		if err != nil {
			return err
		}
	default:
		return common.NewErrorf("unknown action: %s", act)
	}
	return nil
}
