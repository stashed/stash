package api

// ResourceHandler can handle admission requests that happen to a
// resource.
//  * OnCreate is called when an object is created. If an error is
//      returned admission is denied. Otherwise, if an object is
//      returned, it is used to compute a patch and should be used as
//      MutatingAdmissionWebhook.
//  * OnUpdate is called when an object is updated. Note that oldObj is
//      the existing object.  If an error is  returned admission is denied.
//      Otherwise, if an object is returned, it is used to compute a patch
//      and should be used as MutatingAdmissionWebhook.
//  * OnDelete will gets the current state of object when delete request
//      is received.
type ResourceHandler interface {
	OnCreate(obj interface{}) (interface{}, error)
	OnUpdate(oldObj, newObj interface{}) (interface{}, error)
	OnDelete(obj interface{}) error
}

// ResourceHandlerFuncs is an adaptor to let you easily specify as many or
// as few of the notification functions as you want while still implementing
// ResourceHandler.
type ResourceHandlerFuncs struct {
	CreateFunc func(obj interface{}) (interface{}, error)
	UpdateFunc func(oldObj, newObj interface{}) (interface{}, error)
	DeleteFunc func(obj interface{}) error
}

// OnCreate calls CreateFunc if it's not nil.
func (r ResourceHandlerFuncs) OnCreate(obj interface{}) (interface{}, error) {
	if r.CreateFunc != nil {
		return r.CreateFunc(obj)
	}
	return nil, nil
}

// OnUpdate calls UpdateFunc if it's not nil.
func (r ResourceHandlerFuncs) OnUpdate(oldObj, newObj interface{}) (interface{}, error) {
	if r.UpdateFunc != nil {
		return r.UpdateFunc(oldObj, newObj)
	}
	return nil, nil
}

// OnDelete calls DeleteFunc if it's not nil.
func (r ResourceHandlerFuncs) OnDelete(obj interface{}) error {
	if r.DeleteFunc != nil {
		return r.DeleteFunc(obj)
	}
	return nil
}
