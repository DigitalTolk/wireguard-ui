package util

import "sync"

var IPToSubnetRange = map[string]uint16{}
var IPToSubnetRangeMutex sync.RWMutex
var TgUseridToClientID = map[int64][]string{}
var TgUseridToClientIDMutex sync.RWMutex
var DBUsersToCRC32 = map[string]uint32{}
var DBUsersToCRC32Mutex sync.RWMutex
