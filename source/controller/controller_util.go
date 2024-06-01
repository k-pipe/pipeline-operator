package controller

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
)

// called whenever an error occurred, to create an error event
func failed(errormessage string, object runtime.Object, r record.EventRecorder) ctrl.Result {
	r.Event(object, "Warning", "ReconciliationError", errormessage)
	return ctrl.Result{}
}
