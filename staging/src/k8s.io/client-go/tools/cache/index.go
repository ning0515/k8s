/*
Copyright 2014 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cache

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/util/sets"
)

// Indexer extends Store with multiple indices and restricts each
// accumulator to simply hold the current object (and be empty after
// Delete).
//
// There are three kinds of strings here:
//  1. a storage key, as defined in the Store interface,
//  2. a name of an index, and
//  3. an "indexed value", which is produced by an IndexFunc and
//     can be a field value or any other string computed from the object.
type Indexer interface {
	Store
	// Index returns the stored objects whose set of indexed values
	// intersects the set of indexed values of the given object, for
	// the named index

	// Indexers map[string]IndexFunc
	// Indexers是一群索引函数的集合,键是索引器的名字,例如namespace,值是索引函数，例如cache.MetaNamespaceIndexFunc
	// 通过indexers[indexName]获得indexFunc，通过indexFunc(obj)获得indexValues,例如： indexedValues = [kube-system]
	// storeKeySet = index[indexedValues[0]]
	// 通过Indices[indexName]获得对应的Index，最后返回Index[indexValues]中对应的所有资源对象的key
	// 注意indexValues可以为数组
	// indexName 是索引器的名字
	// index最后获得的就是类似&threadSafeMap{items map[string]interface{}: "kube-system/coredns-7569857846-pgq8n"}下面的值
	// 部分情况下也就是命名空间对应的资源名称下面的资源列表,item的键来自于index[indexedValues[0]]
	Index(indexName string, obj interface{}) ([]interface{}, error)
	// IndexKeys returns the storage keys of the stored objects whose
	// set of indexed values for the named index includes the given
	// indexed value
	// 通过Indices[indexName]获得对应的Index，之后获得Index[indexValues]，
	// 并排序得到有序key集合
	// indexFunc := indexers[indexName]
	IndexKeys(indexName, indexedValue string) ([]string, error)
	// ListIndexFuncValues returns all the indexed values of the given index
	// 获得该IndexName对应的所有Index中的index_key集合
	ListIndexFuncValues(indexName string) []string
	// ByIndex returns the stored objects whose set of indexed values
	// for the named index includes the given indexed value
	// 返回Index中对应indexedValue的obj集合
	ByIndex(indexName, indexedValue string) ([]interface{}, error)
	// GetIndexers return the indexers
	// 返回indexers
	GetIndexers() Indexers

	// AddIndexers adds more indexers to this store.  If you call this after you already have data
	// in the store, the results are undefined.
	// 添加Indexer
	AddIndexers(newIndexers Indexers) error
}

// IndexFunc knows how to compute the set of indexed values for an object.
// 如果索引器是namespace的话，索引函数就可以根据具体的obj，计算出对应的命名空间切片，如 indexedValues = [kube-system]
type IndexFunc func(obj interface{}) ([]string, error)

// IndexFuncToKeyFuncAdapter adapts an indexFunc to a keyFunc.  This is only useful if your index function returns
// unique values for every object.  This conversion can create errors when more than one key is found.  You
// should prefer to make proper key and index functions.
func IndexFuncToKeyFuncAdapter(indexFunc IndexFunc) KeyFunc {
	return func(obj interface{}) (string, error) {
		indexKeys, err := indexFunc(obj)
		if err != nil {
			return "", err
		}
		if len(indexKeys) > 1 {
			return "", fmt.Errorf("too many keys: %v", indexKeys)
		}
		if len(indexKeys) == 0 {
			return "", fmt.Errorf("unexpected empty indexKeys")
		}
		return indexKeys[0], nil
	}
}

const (
	// NamespaceIndex is the lookup name for the most common index function, which is to index by the namespace field.
	NamespaceIndex string = "namespace"
)

// MetaNamespaceIndexFunc is a default index function that indexes based on an object's namespace
func MetaNamespaceIndexFunc(obj interface{}) ([]string, error) {
	meta, err := meta.Accessor(obj)
	if err != nil {
		return []string{""}, fmt.Errorf("object has no meta: %v", err)
	}
	return []string{meta.GetNamespace()}, nil
}

// Index maps the indexed value to a set of keys in the store that match on that value
// 举例：key是具体的命名空间,值是namespace/name集合
type Index map[string]sets.String

// Indexers maps a name to an IndexFunc
// 举例：Key是索引器的名字，比如命名空间，值是对应的函数，可以将具体的资源对象所在的命名空间列表返回，返回以后可以给Index当KEY使用
type Indexers map[string]IndexFunc

// Indices maps a name to an Index
// 举例：key是索引器的名字，如namespace，值是根据索引器计算出来的Index
type Indices map[string]Index
