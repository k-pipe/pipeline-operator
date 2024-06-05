package controller

import (
	"context"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// called whenever an error occurred, to create an error event
func failed(errormessage string, object runtime.Object, r record.EventRecorder) ctrl.Result {
	r.Event(object, "Warning", "ReconciliationError", errormessage)
	return ctrl.Result{}
}

// gets a resource, returns bool indication if not found or not, error is nil in case that object was not found
func NotExistsResource(r client.Reader, ctx context.Context, resource client.Object, name types.NamespacedName) (bool, error) {
	err := r.Get(ctx, name, resource)
	if (err != nil) && apierrors.IsNotFound(err) {
		// not found is not considered an error
		return true, nil
	}
	return false, err
}

// Sets the status condition of a resource. The resource's status conditions array is also passed as argument
func SetStatusCondition(writer client.SubResourceWriter, ctx context.Context, resource client.Object, statusConditions *[]metav1.Condition, statusType string, status metav1.ConditionStatus, message string) error {
	log := log.FromContext(ctx)

	if (statusConditions != nil) && meta.IsStatusConditionPresentAndEqual(*statusConditions, statusType, status) {
		// no change in status
		return nil
	}

	log.Info("Updating status " + statusType + " to " + string(status))

	// set the status condition
	meta.SetStatusCondition(
		statusConditions,
		metav1.Condition{
			Type:    statusType,
			Status:  status,
			Reason:  "Reconciling",
			Message: message,
		},
	)

	if err := writer.Update(ctx, resource); err != nil {
		log.Error(err, "Failed to update status "+statusType+" to "+string(status))
		return err
	}
	return nil
}

func NameSpacedName(resource client.Object) types.NamespacedName {
	return types.NamespacedName{Name: resource.GetName(), Namespace: resource.GetNamespace()}
}

func CreateOrUpdate(r client.Reader, w client.Writer, ctx context.Context, resource client.Object, empty client.Object) error {
	notExist, err := NotExistsResource(r, ctx, empty, NameSpacedName(resource))
	if err != nil {
		return err
	}
	if !notExist {
		log.FromContext(ctx).Info("Deleting existing resource: " + resource.GetName())
		if err := w.Delete(ctx, empty); err != nil {
			return err
		}
	}
	return w.Create(ctx, resource)
}
