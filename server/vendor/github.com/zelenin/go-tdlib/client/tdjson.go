//go:build darwin
// +build darwin

package client

/*
#cgo darwin CFLAGS: -I/usr/local/include
#cgo linux LDFLAGS: -L/usr/local/lib -Wl,--start-group -ltdjson_static -ltdjson_private -ltdclient -ltdcore -ltdmtproto -ltdactor -ltdapi -ltddb -ltdsqlite -ltdnet -ltdutils -ltde2e -Wl,--end-group -lstdc++ -lssl -lcrypto -ldl -lz -lm
*/
import "C"
