package routers

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context/param"
)

func init() {
    const KubeJobController = "wayne/src/backend/controllers/kubernetes/job:KubeJobController"
    beego.GlobalControllerRouter[KubeJobController] = append(
        beego.GlobalControllerRouter[KubeJobController],
        beego.ControllerComments{
            Method: "ListJobByCronJob",
            Router: `/namespaces/:namespace/clusters/:cluster`,
            AllowHTTPMethods: []string{"get"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil},
        beego.ControllerComments{
            Method: "DeleteJob",
            Router: `/namespaces/:namespace/clusters/:cluster`,
            AllowHTTPMethods: []string{"delete"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

}
