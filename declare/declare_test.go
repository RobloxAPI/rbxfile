package declare_test

import (
	"fmt"

	. "github.com/robloxapi/rbxfile/declare"
)

func Example() {
	root := Root{
		Instance("Part", Ref("RBX12345678"),
			Property("Name", String, "BasePlate"),
			Property("CanCollide", Bool, true),
			Property("Position", Vector3, 0, 10, 0),
			Property("Size", Vector3, 2, 1.2, 4),
			Instance("CFrameValue",
				Property("Name", String, "Value"),
				Property("Value", CFrame, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 1),
			),
			Instance("ObjectValue",
				Property("Name", String, "Value"),
				Property("Value", Reference, "RBX12345678"),
			),
		),
	}.Declare()
	fmt.Println(root)
}
