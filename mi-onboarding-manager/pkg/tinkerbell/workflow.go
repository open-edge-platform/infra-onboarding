package tinkerbell

import (
	tink "github.com/tinkerbell/tink/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewWorkflow(name, ns, mac string) *tink.Workflow {
	wf := &tink.Workflow{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Workflow",
			APIVersion: "tinkerbell.org/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: tink.WorkflowSpec{
			HardwareMap: map[string]string{
				"device_1": mac,
			},
		},
	}

	return wf
}
