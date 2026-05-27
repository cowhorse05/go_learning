package v1

import "k8s.io/apimachinery/pkg/runtime"

func (in *SimpleApp) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(SimpleApp)
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.TypeMeta = out.TypeMeta
	out.Spec = in.Spec
	out.Status = in.Status
	return out
}

func (in *SimpleApp) DeepCopyInto(out *SimpleApp) {
	*out = *in
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
}

func (in *SimpleAppList) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(SimpleAppList)
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		out.Items = make([]SimpleApp, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}
	return out
}
