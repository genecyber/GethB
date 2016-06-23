function checkRequest(r) {
    // Only allow GET requests to the API
    return r.Method == "GET";
}

function init() {
	return "Cap'n Proto Loaded"
}