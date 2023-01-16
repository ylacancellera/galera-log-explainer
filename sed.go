package main

func sedByName(ni NodeInfo) []string {
	if len(ni.NodeNames) == 0 {
		return nil
	}
	elem := ni.NodeNames[0]
	args := sedSliceWith(ni.NodeUUIDs, elem)
	args = append(args, sedSliceWith(ni.IPs, elem)...)
	return args
}

func sedByIP(ni NodeInfo) []string {
	if len(ni.IPs) == 0 {
		return nil
	}
	elem := ni.IPs[0]
	args := sedSliceWith(ni.NodeUUIDs, elem)
	args = append(args, sedSliceWith(ni.NodeNames, elem)...)
	return args
}

func sedSliceWith(elems []string, replace string) []string {
	args := []string{}
	for _, elem := range elems {
		args = append(args, "-e")
		args = append(args, "s/"+elem+"/"+replace+"/g")
	}
	return args
}
