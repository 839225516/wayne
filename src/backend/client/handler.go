// Proxy is a package responsible for doing common operations on kubernetes resources
// like UPDATE DELETE CREATE GET deployment and so on.
package client

import (
	"context"
	"encoding/json"
	"fmt"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"

	"wayne/src/backend/client/api"
	"wayne/src/backend/util/logs"
)

type ResourceHandler interface {
	Create(kind string, namespace string, object *runtime.Unknown) (*runtime.Unknown, error)
	Update(kind string, namespace string, name string, object *runtime.Unknown) (*runtime.Unknown, error)
	Get(kind string, namespace string, name string) (runtime.Object, error)
	List(kind string, namespace string, labelSelector string) ([]runtime.Object, error)
	Delete(kind string, namespace string, name string, options *meta_v1.DeleteOptions) error
}

type resourceHandler struct {
	client       *kubernetes.Clientset
	cacheFactory *CacheFactory
}

func NewResourceHandler(kubeClient *kubernetes.Clientset, cacheFactory *CacheFactory) ResourceHandler {
	return &resourceHandler{
		client:       kubeClient,
		cacheFactory: cacheFactory,
	}
}

func (h *resourceHandler) Create(kind string, namespace string, object *runtime.Unknown) (*runtime.Unknown, error) {
	resource, err := h.getResource(kind)
	if err != nil {
		return nil, err
	}

	kubeClient := h.getClientByGroupVersion(resource.GroupVersionResourceKind.GroupVersionResource)
	req := kubeClient.Post().
		Resource(kind).
		SetHeader("Content-Type", "application/json").
		Body([]byte(object.Raw))
	if resource.Namespaced {
		req.Namespace(namespace)
	}
	var result runtime.Unknown
	err = req.Do(context.TODO()).Into(&result)

	return &result, err
}

func (h *resourceHandler) Update(kind string, namespace string, name string, object *runtime.Unknown) (*runtime.Unknown, error) {

	resource, err := h.getResource(kind)
	if err != nil {
		return nil, err
	}

	kubeClient := h.getClientByGroupVersion(resource.GroupVersionResourceKind.GroupVersionResource)

	// Kubernetes 中的所有资源对象，都有一个全局唯一的版本号（metadata.resourceVersion）
	// kube-apiserver 会校验用户 update 请求提交对象中的 resourceVersion 一定要和当前 K8s 中这个对象最新的 resourceVersion 一致，
	// 才能接受本次 update
	// 更新services时没有带ResourceVersion 会导致更新失败
	if kind == "services" {
		getReq := kubeClient.Get().Resource(kind).Name(name).Namespace(namespace)
		var result1 runtime.Unknown
		//fmt.Println("查service")
		//err := getReq.Do(context.TODO()).Into(&result1)
		err := getReq.Do(context.TODO()).Into(&result1)
		if err != nil {
			fmt.Println(err)
		}
		//fmt.Println(string(result1.Raw))
		var serviceOld, seviceNew v1.Service
		err = json.Unmarshal(result1.Raw, &serviceOld)
		if err != nil {
			fmt.Println("servcieOld json解释失败")
		}

		err = json.Unmarshal(object.Raw, &seviceNew)
		if err != nil {
			fmt.Println("servcieNew json解释失败")
		}

		seviceNew.ResourceVersion = serviceOld.ResourceVersion

		object.Raw, err = json.Marshal(seviceNew)
		if err != nil {
			fmt.Println("servcieNew2json失败")
		}
		fmt.Println("serviceNew json:", string(object.Raw))
	}

	req := kubeClient.Put().
		Resource(kind).
		Name(name).
		SetHeader("Content-Type", "application/json").
		Body([]byte(object.Raw))
	if resource.Namespaced {
		req.Namespace(namespace)
	}

	var result runtime.Unknown
	err = req.Do(context.TODO()).Into(&result)

	return &result, err
}

func (h *resourceHandler) Delete(kind string, namespace string, name string, options *meta_v1.DeleteOptions) error {

	resource, err := h.getResource(kind)
	if err != nil {
		return err
	}

	kubeClient := h.getClientByGroupVersion(resource.GroupVersionResourceKind.GroupVersionResource)
	req := kubeClient.Delete().
		Resource(kind).
		Name(name).
		Body(options)
	if resource.Namespaced {
		req.Namespace(namespace)
	}

	return req.Do(context.TODO()).Error()
}

// Get object from cache
func (h *resourceHandler) Get(kind string, namespace string, name string) (runtime.Object, error) {
	resource, err := h.getResource(kind)
	if err != nil {
		return nil, err
	}

	genericInformer, err := h.cacheFactory.sharedInformerFactory.ForResource(resource.GroupVersionResourceKind.GroupVersionResource)
	if err != nil {
		return nil, err
	}
	lister := genericInformer.Lister()
	var result runtime.Object
	if resource.Namespaced {
		result, err = lister.ByNamespace(namespace).Get(name)
		if err != nil {
			return nil, err
		}
	} else {
		result, err = lister.Get(name)
		if err != nil {
			return nil, err
		}
	}
	result.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
		Group:   resource.GroupVersionResourceKind.Group,
		Version: resource.GroupVersionResourceKind.Version,
		Kind:    resource.GroupVersionResourceKind.Kind,
	})

	return result, nil
}

// Get object from cache
func (h *resourceHandler) List(kind string, namespace string, labelSelector string) ([]runtime.Object, error) {

	resource, err := h.getResource(kind)
	if err != nil {
		return nil, err
	}

	genericInformer, err := h.cacheFactory.sharedInformerFactory.ForResource(resource.GroupVersionResourceKind.GroupVersionResource)
	if err != nil {
		return nil, err
	}
	selectors, err := labels.Parse(labelSelector)
	if err != nil {
		logs.Error("Build label selector error.", err)
		return nil, err
	}

	lister := genericInformer.Lister()
	var objs []runtime.Object
	if resource.Namespaced {
		objs, err = lister.ByNamespace(namespace).List(selectors)
		if err != nil {
			return nil, err
		}
	} else {
		objs, err = lister.List(selectors)
		if err != nil {
			return nil, err
		}
	}

	for i := range objs {
		objs[i].GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
			Group:   resource.GroupVersionResourceKind.Group,
			Version: resource.GroupVersionResourceKind.Version,
			Kind:    resource.GroupVersionResourceKind.Kind,
		})
	}

	return objs, nil
}

func (h *resourceHandler) getResource(kind string) (resource api.ResourceMap, err error) {
	resourceMap, err := api.GetResourceMap(h.client)
	if err != nil {
		return
	}
	resource, ok := resourceMap[kind]
	if !ok {
		err = fmt.Errorf("Resource kind (%s) not support yet . ", kind)
		return
	}
	return
}
