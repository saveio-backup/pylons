// defines interfaces to be called by http
package rpc

func Test(params []interface{}) map[string]interface{} {
	return responseSuccess("test")
}
