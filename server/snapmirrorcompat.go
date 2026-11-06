package server

import (
	"fmt"
	"strings"

	"github.com/netapp/ontap-mcp/ontap"
)

// xdpReleaseOrder and xdpCompatibilityRows reproduce the directional unified
// replication (XDP) matrix published by NetApp on 2026-04-17:
// https://docs.netapp.com/us-en/ontap/data-protection/compatible-ontap-versions-snapmirror-concept.html
var xdpReleaseOrder = []string{
	"9.19.1", "9.18.1", "9.17.1", "9.16.1", "9.16.0", "9.15.1", "9.15.0", "9.14.1",
	"9.14.0", "9.13.1", "9.13.0", "9.12.1", "9.12.0", "9.11.1", "9.11.0", "9.10.1",
	"9.10.0", "9.9.1", "9.9.0", "9.8", "9.7", "9.6", "9.5",
}

var xdpCompatibilityRows = map[string]string{
	"9.19.1": "11111 11100 00000 00000 000",
	"9.18.1": "11111 11101 10000 00000 000",
	"9.17.1": "11111 11111 11000 00000 000",
	"9.16.1": "11111 11111 11110 00000 000",
	"9.16.0": "11111 11101 01010 10000 000",
	"9.15.1": "11111 11111 11111 11100 000",
	"9.15.0": "11110 11101 01010 10100 000",
	"9.14.1": "11111 11111 11111 11111 000",
	"9.14.0": "00110 10111 01010 10100 000",
	"9.13.1": "01111 11111 11111 11111 000",
	"9.13.0": "00000 10101 11010 10101 000",
	"9.12.1": "00111 11111 11111 11111 100",
	"9.12.0": "00010 10101 01110 10101 100",
	"9.11.1": "00011 11111 11111 11111 110",
	"9.11.0": "00000 10101 01011 10101 110",
	"9.10.1": "00001 11111 11111 11111 111",
	"9.10.0": "00000 10101 01010 11101 111",
	"9.9.1":  "00000 11111 11111 11111 111",
	"9.9.0":  "00000 00101 01010 10111 111",
	"9.8":    "00000 00001 11111 11111 111",
	"9.7":    "00000 00000 01111 11111 111",
	"9.6":    "00000 00000 00011 11111 111",
	"9.5":    "00000 00000 00000 11111 111",
}

type compatibilityResult struct {
	Compatible         bool
	Known              bool
	SourceRelease      string
	DestinationRelease string
	Reason             string
}

func checkXDPCompatibility(source, destination ontap.Remote) compatibilityResult {
	result := compatibilityResult{
		SourceRelease:      ontapRelease(source.Version),
		DestinationRelease: ontapRelease(destination.Version),
	}

	if source.Model == ontap.ASAr2 && source.Model != destination.Model {
		result.Known = true
		result.Reason = "ASA r2 systems support SnapMirror relationships only with other ASA r2 systems"
		return result
	}

	row, sourceKnown := xdpCompatibilityRows[result.SourceRelease]
	destinationIndex := releaseIndex(result.DestinationRelease)
	if !sourceKnown || destinationIndex < 0 {
		result.Reason = fmt.Sprintf(
			"version compatibility is not available in the bundled XDP matrix for source %s and destination %s",
			result.SourceRelease, result.DestinationRelease,
		)
		return result
	}

	row = strings.ReplaceAll(row, " ", "")
	if len(row) != len(xdpReleaseOrder) {
		result.Reason = fmt.Sprintf("bundled XDP compatibility row for %s is invalid", result.SourceRelease)
		return result
	}

	result.Known = true
	result.Compatible = row[destinationIndex] == '1'
	if result.Compatible {
		result.Reason = fmt.Sprintf("ONTAP %s to ONTAP %s is supported for asynchronous XDP replication", result.SourceRelease, result.DestinationRelease)
	} else {
		result.Reason = fmt.Sprintf("ONTAP %s to ONTAP %s is not supported for asynchronous XDP replication", result.SourceRelease, result.DestinationRelease)
	}
	return result
}

func ontapRelease(version ontap.Version) string {
	if version.Generation == 9 && version.Major <= 8 && version.Minor == 0 {
		return fmt.Sprintf("%d.%d", version.Generation, version.Major)
	}
	return fmt.Sprintf("%d.%d.%d", version.Generation, version.Major, version.Minor)
}

func releaseIndex(release string) int {
	for index, candidate := range xdpReleaseOrder {
		if candidate == release {
			return index
		}
	}
	return -1
}
