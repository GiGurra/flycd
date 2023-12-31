package model

import "time"

type VolumeState struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	SizeGb    int       `json:"size_gb"`
	State     string    `json:"state"`
	Region    string    `json:"region"`
	Encrypted bool      `json:"encrypted"`
	CreatedAt time.Time `json:"created_at"`

	// TODO: later:
	//  App               App          `json:"App"`
	//  Snapshots         Snapshots    `json:"Snapshots"`
	//  AttachedAllocation interface{} `json:"AttachedAllocation"`
	//  AttachedMachine    Machine     `json:"AttachedMachine"`
	//  Host               Host        `json:"Host"`
}

type VolumeConfig struct {
	Name   string `json:"name" yaml:"name"`
	SizeGb int    `json:"size_gb" yaml:"size_gb"`
	Count  int    `json:"count" yaml:"count"`

	// TODO: later:
	//  App               App          `json:"App"`
	//  Snapshots         Snapshots    `json:"Snapshots"`
	//  AttachedAllocation interface{} `json:"AttachedAllocation"`
	//  AttachedMachine    Machine     `json:"AttachedMachine"`
	//  Host               Host        `json:"Host"`
}

/*type App struct {
	Name            string `json:"Name"`
	PlatformVersion string `json:"PlatformVersion"`
}*/

/*type Snapshots struct {
	Nodes interface{} `json:"Nodes"`
}*/

/*type Machine struct {
	ID     string      `json:"ID"`
	Name   string      `json:"Name"`
	State  string      `json:"State"`
	Region string      `json:"Region"`
	Config MachineConf `json:"Config"`
	App    interface{} `json:"App"`
	IPs    IPs         `json:"IPs"`
}
*/

/*type MachineConf struct {
	Init    interface{} `json:"init"`
	Restart interface{} `json:"restart"`
}*/

/*type IPs struct {
	Nodes interface{} `json:"Nodes"`
}*/

/*type Host struct {
	ID string `json:"ID"`
}*/

/**
[
    {
        "id": "vol_nylzre2p7j3rqmkp",
        "App": {
            "Name": "",
            "PlatformVersion": "machines"
        },
        "Name": "ravendb",
        "SizeGb": 10,
        "Snapshots": {
            "Nodes": null
        },
        "State": "created",
        "Region": "arn",
        "Encrypted": true,
        "CreatedAt": "2023-07-02T20:00:41Z",
        "AttachedAllocation": null,
        "AttachedMachine": {
            "ID": "6e82dd74f09558",
            "Name": "green-bush-9546",
            "State": "",
            "Region": "",
            "Config": {
                "init": {},
                "restart": {}
            },
            "App": null,
            "IPs": {
                "Nodes": null
            }
        },
        "Host": {
            "ID": "f1ec"
        }
    }
]
*/
