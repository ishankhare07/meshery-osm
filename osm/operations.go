package osm

import (
	"context"
	"fmt"

	"github.com/layer5io/meshery-adapter-library/adapter"
	"github.com/layer5io/meshery-adapter-library/common"
	"github.com/layer5io/meshery-adapter-library/status"
	internalconfig "github.com/layer5io/meshery-osm/internal/config"
	"github.com/layer5io/meshkit/errors"
)

func (h *Handler) ApplyOperation(ctx context.Context, request adapter.OperationRequest) error {
	operations := make(adapter.Operations)
	err := h.Config.GetObject(adapter.OperationsKey, &operations)
	if err != nil {
		return err
	}

	e := &adapter.Event{
		Operationid: request.OperationID,
		Summary:     status.Deploying,
		Details:     "Operation is not supported",
	}

	//deployment
	switch request.OperationName {
	case internalconfig.OSMOperation:
		go func(hh *Handler, adapterEvent *adapter.Event) {
			version := string(operations[request.OperationName].Versions[0])
			stat, err := hh.Execute(request.IsDeleteOperation, version, request.Namespace)
			if err != nil {
				adapterEvent.Summary = fmt.Sprintf("Error while %s OSM service mesh", stat)
				adapterEvent.Details = err.Error()
				hh.StreamErr(adapterEvent, err)
				return
			}
			adapterEvent.Summary = fmt.Sprintf("OSM service mesh %s successfully", stat)
			adapterEvent.Details = fmt.Sprintf("The OSM service mesh is now %s.", stat)
			hh.StreamInfo(adapterEvent)
		}(h, e)
	case internalconfig.OSMSampleBookBuyer:
		go func(handler *Handler, adapterEvent *adapter.Event) {
			version := string(operations[request.OperationName].Versions[0])
			stat, err := handler.OSMSampleBookBuyerExecute(request.IsDeleteOperation, version)
			if err != nil {
				adapterEvent.Summary = fmt.Sprintf("Error while %s %s application", stat, request.OperationName)
				adapterEvent.Details = err.Error()
				handler.StreamErr(adapterEvent, err)
				return
			}
			adapterEvent.Summary = fmt.Sprintf("OSM sample book buyer app %s successfully", stat)
			adapterEvent.Details = fmt.Sprintf("The OSM book buyer sample application is now %s.", stat)
			handler.StreamInfo(adapterEvent)
		}(h, e)
	case common.BookInfoOperation, common.HTTPBinOperation, common.ImageHubOperation, common.EmojiVotoOperation:
		go func(hh *Handler, ee *adapter.Event) {
			appName := operations[request.OperationName].AdditionalProperties[common.ServiceName]
			stat, err := hh.installSampleApp(request.IsDeleteOperation, request.Namespace, operations[request.OperationName].Templates)
			if err != nil {
				e.Summary = fmt.Sprintf("Error while %s %s application", stat, appName)
				e.Details = err.Error()
				hh.StreamErr(e, err)
				return
			}
			ee.Summary = fmt.Sprintf("%s application %s successfully", appName, stat)
			ee.Details = fmt.Sprintf("The %s application is now %s.", appName, stat)
			hh.StreamInfo(e)
		}(h, e)
	case common.SmiConformanceOperation:
		go func(hh *Handler, ee *adapter.Event) {
			name := operations[request.OperationName].Description
			err := hh.ValidateSMIConformance(&adapter.SmiTestOptions{
				Ctx:  context.TODO(),
				OpID: ee.Operationid,
			})
			if err != nil {
				e.Summary = fmt.Sprintf("Error while %s %s test", status.Running, name)
				e.Details = err.Error()
				hh.StreamErr(e, err)
				return
			}
			ee.Summary = fmt.Sprintf("%s test %s successfully", name, status.Completed)
			ee.Details = ""
			hh.StreamInfo(e)
		}(h, e)
	default:
		h.StreamErr(e, errors.NewDefault(errors.ErrOpInvalid, "Invalid operation"))
	}
	return nil
}
