package xrayclient

// Typical use:
//
//	client, err := xrayclient.New(configJSON)
//	if err != nil { return err }
//	defer client.Close()
//
//	proxiedHTTP := client.HTTPClient(30 * time.Second)
//	// Only requests made by proxiedHTTP pass through Xray.
//	response, err := proxiedHTTP.Get("https://example.com")
//
// An adapter that must connect directly should keep its normal http.Client.
