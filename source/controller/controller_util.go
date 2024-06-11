package controller

import (
	"context"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// gets a resource, returns bool indication if not found or not, error is nil in case that object was not found
func NotExistsResource(r client.Reader, ctx context.Context, resource client.Object, name types.NamespacedName) (bool, error) {
	err := r.Get(ctx, name, resource)
	if (err != nil) && apierrors.IsNotFound(err) {
		// not found is not considered an error
		return true, nil
	}
	return false, err
}

const LOG_WIDTH = 80

// Sets the status condition of a resource. The resource's status conditions array is also passed as argument
func SetStatusCondition(writer client.SubResourceWriter, ctx context.Context, log func(string, ...interface{}), resource client.Object, statusConditions *[]metav1.Condition, statusType string, status metav1.ConditionStatus, message string) error {
	if (statusConditions != nil) && meta.IsStatusConditionPresentAndEqual(*statusConditions, statusType, status) {
		// no change in status
		return nil
	}

	log("Updating status " + statusType + " to " + string(status))

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

	return writer.Update(ctx, resource)
}

func NameSpacedName(resource client.Object) types.NamespacedName {
	return types.NamespacedName{Name: resource.GetName(), Namespace: resource.GetNamespace()}
}

func CreateOrUpdate(r client.Reader, w client.Writer, ctx context.Context, log func(string, ...interface{}), resource client.Object, empty client.Object) error {
	notExist, err := NotExistsResource(r, ctx, empty, NameSpacedName(resource))
	if err != nil {
		return err
	}
	if notExist {
		log("Creating resource: " + resource.GetName())
		return w.Create(ctx, resource)
	} else {
		log("Updating existing resource: " + resource.GetName())
		return w.Update(ctx, resource)
	}
}

func Logger(ctx context.Context, req ctrl.Request, prefix string) func(string, ...interface{}) {
	l := log.FromContext(ctx)
	res := func(message string, keysAndValues ...interface{}) {
		line := "[" + prefix + "]    " + message
		l.Info(line+repeat(" ", min0(LOG_WIDTH-len(line))), keysAndValues...)
	}
	res(repeat(",", LOG_WIDTH-6-len(prefix)))
	name := req.Namespace + ":" + req.Name
	res(repeat(" ", min0((LOG_WIDTH-len(name)-6-len(prefix))/2)) + name)
	return res
}

func min0(i int) int {
	if i > 0 {
		return i
	} else {
		return 0
	}
}

func LoggingDone(log func(string, ...interface{})) {
	log(repeat("'", LOG_WIDTH-8))
}

func repeat(s string, num int) string {
	res := ""
	for i := 0; i < num; i++ {
		res = res + s
	}
	return res
}
