// Code generated by counterfeiter. DO NOT EDIT.
package mocks

import (
	"context"
	"sync"

	"github.com/alphagov/paas-elasticache-broker/providers"
)

type FakeProvider struct {
	DeleteCacheParameterGroupStub        func(context.Context, string) error
	deleteCacheParameterGroupMutex       sync.RWMutex
	deleteCacheParameterGroupArgsForCall []struct {
		arg1 context.Context
		arg2 string
	}
	deleteCacheParameterGroupReturns struct {
		result1 error
	}
	deleteCacheParameterGroupReturnsOnCall map[int]struct {
		result1 error
	}
	DeprovisionStub        func(context.Context, string, providers.DeprovisionParameters) error
	deprovisionMutex       sync.RWMutex
	deprovisionArgsForCall []struct {
		arg1 context.Context
		arg2 string
		arg3 providers.DeprovisionParameters
	}
	deprovisionReturns struct {
		result1 error
	}
	deprovisionReturnsOnCall map[int]struct {
		result1 error
	}
	FindSnapshotsStub        func(context.Context, string) ([]providers.SnapshotInfo, error)
	findSnapshotsMutex       sync.RWMutex
	findSnapshotsArgsForCall []struct {
		arg1 context.Context
		arg2 string
	}
	findSnapshotsReturns struct {
		result1 []providers.SnapshotInfo
		result2 error
	}
	findSnapshotsReturnsOnCall map[int]struct {
		result1 []providers.SnapshotInfo
		result2 error
	}
	GenerateCredentialsStub        func(context.Context, string, string) (*providers.Credentials, error)
	generateCredentialsMutex       sync.RWMutex
	generateCredentialsArgsForCall []struct {
		arg1 context.Context
		arg2 string
		arg3 string
	}
	generateCredentialsReturns struct {
		result1 *providers.Credentials
		result2 error
	}
	generateCredentialsReturnsOnCall map[int]struct {
		result1 *providers.Credentials
		result2 error
	}
	GetInstanceParametersStub        func(context.Context, string) (providers.InstanceParameters, error)
	getInstanceParametersMutex       sync.RWMutex
	getInstanceParametersArgsForCall []struct {
		arg1 context.Context
		arg2 string
	}
	getInstanceParametersReturns struct {
		result1 providers.InstanceParameters
		result2 error
	}
	getInstanceParametersReturnsOnCall map[int]struct {
		result1 providers.InstanceParameters
		result2 error
	}
	GetInstanceTagsStub        func(context.Context, string) (map[string]string, error)
	getInstanceTagsMutex       sync.RWMutex
	getInstanceTagsArgsForCall []struct {
		arg1 context.Context
		arg2 string
	}
	getInstanceTagsReturns struct {
		result1 map[string]string
		result2 error
	}
	getInstanceTagsReturnsOnCall map[int]struct {
		result1 map[string]string
		result2 error
	}
	ProgressStateStub        func(context.Context, string, string, string) (providers.ServiceState, string, error)
	progressStateMutex       sync.RWMutex
	progressStateArgsForCall []struct {
		arg1 context.Context
		arg2 string
		arg3 string
		arg4 string
	}
	progressStateReturns struct {
		result1 providers.ServiceState
		result2 string
		result3 error
	}
	progressStateReturnsOnCall map[int]struct {
		result1 providers.ServiceState
		result2 string
		result3 error
	}
	ProvisionStub        func(context.Context, string, providers.ProvisionParameters) error
	provisionMutex       sync.RWMutex
	provisionArgsForCall []struct {
		arg1 context.Context
		arg2 string
		arg3 providers.ProvisionParameters
	}
	provisionReturns struct {
		result1 error
	}
	provisionReturnsOnCall map[int]struct {
		result1 error
	}
	RevokeCredentialsStub        func(context.Context, string, string) error
	revokeCredentialsMutex       sync.RWMutex
	revokeCredentialsArgsForCall []struct {
		arg1 context.Context
		arg2 string
		arg3 string
	}
	revokeCredentialsReturns struct {
		result1 error
	}
	revokeCredentialsReturnsOnCall map[int]struct {
		result1 error
	}
	StartFailoverTestStub        func(context.Context, string) (string, error)
	startFailoverTestMutex       sync.RWMutex
	startFailoverTestArgsForCall []struct {
		arg1 context.Context
		arg2 string
	}
	startFailoverTestReturns struct {
		result1 string
		result2 error
	}
	startFailoverTestReturnsOnCall map[int]struct {
		result1 string
		result2 error
	}
	UpdateParamGroupParametersStub        func(context.Context, string, providers.UpdateParamGroupParameters) error
	updateParamGroupParametersMutex       sync.RWMutex
	updateParamGroupParametersArgsForCall []struct {
		arg1 context.Context
		arg2 string
		arg3 providers.UpdateParamGroupParameters
	}
	updateParamGroupParametersReturns struct {
		result1 error
	}
	updateParamGroupParametersReturnsOnCall map[int]struct {
		result1 error
	}
	UpdateReplicationGroupStub        func(context.Context, string, providers.UpdateReplicationGroupParameters) error
	updateReplicationGroupMutex       sync.RWMutex
	updateReplicationGroupArgsForCall []struct {
		arg1 context.Context
		arg2 string
		arg3 providers.UpdateReplicationGroupParameters
	}
	updateReplicationGroupReturns struct {
		result1 error
	}
	updateReplicationGroupReturnsOnCall map[int]struct {
		result1 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeProvider) DeleteCacheParameterGroup(arg1 context.Context, arg2 string) error {
	fake.deleteCacheParameterGroupMutex.Lock()
	ret, specificReturn := fake.deleteCacheParameterGroupReturnsOnCall[len(fake.deleteCacheParameterGroupArgsForCall)]
	fake.deleteCacheParameterGroupArgsForCall = append(fake.deleteCacheParameterGroupArgsForCall, struct {
		arg1 context.Context
		arg2 string
	}{arg1, arg2})
	stub := fake.DeleteCacheParameterGroupStub
	fakeReturns := fake.deleteCacheParameterGroupReturns
	fake.recordInvocation("DeleteCacheParameterGroup", []interface{}{arg1, arg2})
	fake.deleteCacheParameterGroupMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeProvider) DeleteCacheParameterGroupCallCount() int {
	fake.deleteCacheParameterGroupMutex.RLock()
	defer fake.deleteCacheParameterGroupMutex.RUnlock()
	return len(fake.deleteCacheParameterGroupArgsForCall)
}

func (fake *FakeProvider) DeleteCacheParameterGroupCalls(stub func(context.Context, string) error) {
	fake.deleteCacheParameterGroupMutex.Lock()
	defer fake.deleteCacheParameterGroupMutex.Unlock()
	fake.DeleteCacheParameterGroupStub = stub
}

func (fake *FakeProvider) DeleteCacheParameterGroupArgsForCall(i int) (context.Context, string) {
	fake.deleteCacheParameterGroupMutex.RLock()
	defer fake.deleteCacheParameterGroupMutex.RUnlock()
	argsForCall := fake.deleteCacheParameterGroupArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeProvider) DeleteCacheParameterGroupReturns(result1 error) {
	fake.deleteCacheParameterGroupMutex.Lock()
	defer fake.deleteCacheParameterGroupMutex.Unlock()
	fake.DeleteCacheParameterGroupStub = nil
	fake.deleteCacheParameterGroupReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeProvider) DeleteCacheParameterGroupReturnsOnCall(i int, result1 error) {
	fake.deleteCacheParameterGroupMutex.Lock()
	defer fake.deleteCacheParameterGroupMutex.Unlock()
	fake.DeleteCacheParameterGroupStub = nil
	if fake.deleteCacheParameterGroupReturnsOnCall == nil {
		fake.deleteCacheParameterGroupReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.deleteCacheParameterGroupReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeProvider) Deprovision(arg1 context.Context, arg2 string, arg3 providers.DeprovisionParameters) error {
	fake.deprovisionMutex.Lock()
	ret, specificReturn := fake.deprovisionReturnsOnCall[len(fake.deprovisionArgsForCall)]
	fake.deprovisionArgsForCall = append(fake.deprovisionArgsForCall, struct {
		arg1 context.Context
		arg2 string
		arg3 providers.DeprovisionParameters
	}{arg1, arg2, arg3})
	stub := fake.DeprovisionStub
	fakeReturns := fake.deprovisionReturns
	fake.recordInvocation("Deprovision", []interface{}{arg1, arg2, arg3})
	fake.deprovisionMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2, arg3)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeProvider) DeprovisionCallCount() int {
	fake.deprovisionMutex.RLock()
	defer fake.deprovisionMutex.RUnlock()
	return len(fake.deprovisionArgsForCall)
}

func (fake *FakeProvider) DeprovisionCalls(stub func(context.Context, string, providers.DeprovisionParameters) error) {
	fake.deprovisionMutex.Lock()
	defer fake.deprovisionMutex.Unlock()
	fake.DeprovisionStub = stub
}

func (fake *FakeProvider) DeprovisionArgsForCall(i int) (context.Context, string, providers.DeprovisionParameters) {
	fake.deprovisionMutex.RLock()
	defer fake.deprovisionMutex.RUnlock()
	argsForCall := fake.deprovisionArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3
}

func (fake *FakeProvider) DeprovisionReturns(result1 error) {
	fake.deprovisionMutex.Lock()
	defer fake.deprovisionMutex.Unlock()
	fake.DeprovisionStub = nil
	fake.deprovisionReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeProvider) DeprovisionReturnsOnCall(i int, result1 error) {
	fake.deprovisionMutex.Lock()
	defer fake.deprovisionMutex.Unlock()
	fake.DeprovisionStub = nil
	if fake.deprovisionReturnsOnCall == nil {
		fake.deprovisionReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.deprovisionReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeProvider) FindSnapshots(arg1 context.Context, arg2 string) ([]providers.SnapshotInfo, error) {
	fake.findSnapshotsMutex.Lock()
	ret, specificReturn := fake.findSnapshotsReturnsOnCall[len(fake.findSnapshotsArgsForCall)]
	fake.findSnapshotsArgsForCall = append(fake.findSnapshotsArgsForCall, struct {
		arg1 context.Context
		arg2 string
	}{arg1, arg2})
	stub := fake.FindSnapshotsStub
	fakeReturns := fake.findSnapshotsReturns
	fake.recordInvocation("FindSnapshots", []interface{}{arg1, arg2})
	fake.findSnapshotsMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeProvider) FindSnapshotsCallCount() int {
	fake.findSnapshotsMutex.RLock()
	defer fake.findSnapshotsMutex.RUnlock()
	return len(fake.findSnapshotsArgsForCall)
}

func (fake *FakeProvider) FindSnapshotsCalls(stub func(context.Context, string) ([]providers.SnapshotInfo, error)) {
	fake.findSnapshotsMutex.Lock()
	defer fake.findSnapshotsMutex.Unlock()
	fake.FindSnapshotsStub = stub
}

func (fake *FakeProvider) FindSnapshotsArgsForCall(i int) (context.Context, string) {
	fake.findSnapshotsMutex.RLock()
	defer fake.findSnapshotsMutex.RUnlock()
	argsForCall := fake.findSnapshotsArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeProvider) FindSnapshotsReturns(result1 []providers.SnapshotInfo, result2 error) {
	fake.findSnapshotsMutex.Lock()
	defer fake.findSnapshotsMutex.Unlock()
	fake.FindSnapshotsStub = nil
	fake.findSnapshotsReturns = struct {
		result1 []providers.SnapshotInfo
		result2 error
	}{result1, result2}
}

func (fake *FakeProvider) FindSnapshotsReturnsOnCall(i int, result1 []providers.SnapshotInfo, result2 error) {
	fake.findSnapshotsMutex.Lock()
	defer fake.findSnapshotsMutex.Unlock()
	fake.FindSnapshotsStub = nil
	if fake.findSnapshotsReturnsOnCall == nil {
		fake.findSnapshotsReturnsOnCall = make(map[int]struct {
			result1 []providers.SnapshotInfo
			result2 error
		})
	}
	fake.findSnapshotsReturnsOnCall[i] = struct {
		result1 []providers.SnapshotInfo
		result2 error
	}{result1, result2}
}

func (fake *FakeProvider) GenerateCredentials(arg1 context.Context, arg2 string, arg3 string) (*providers.Credentials, error) {
	fake.generateCredentialsMutex.Lock()
	ret, specificReturn := fake.generateCredentialsReturnsOnCall[len(fake.generateCredentialsArgsForCall)]
	fake.generateCredentialsArgsForCall = append(fake.generateCredentialsArgsForCall, struct {
		arg1 context.Context
		arg2 string
		arg3 string
	}{arg1, arg2, arg3})
	stub := fake.GenerateCredentialsStub
	fakeReturns := fake.generateCredentialsReturns
	fake.recordInvocation("GenerateCredentials", []interface{}{arg1, arg2, arg3})
	fake.generateCredentialsMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2, arg3)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeProvider) GenerateCredentialsCallCount() int {
	fake.generateCredentialsMutex.RLock()
	defer fake.generateCredentialsMutex.RUnlock()
	return len(fake.generateCredentialsArgsForCall)
}

func (fake *FakeProvider) GenerateCredentialsCalls(stub func(context.Context, string, string) (*providers.Credentials, error)) {
	fake.generateCredentialsMutex.Lock()
	defer fake.generateCredentialsMutex.Unlock()
	fake.GenerateCredentialsStub = stub
}

func (fake *FakeProvider) GenerateCredentialsArgsForCall(i int) (context.Context, string, string) {
	fake.generateCredentialsMutex.RLock()
	defer fake.generateCredentialsMutex.RUnlock()
	argsForCall := fake.generateCredentialsArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3
}

func (fake *FakeProvider) GenerateCredentialsReturns(result1 *providers.Credentials, result2 error) {
	fake.generateCredentialsMutex.Lock()
	defer fake.generateCredentialsMutex.Unlock()
	fake.GenerateCredentialsStub = nil
	fake.generateCredentialsReturns = struct {
		result1 *providers.Credentials
		result2 error
	}{result1, result2}
}

func (fake *FakeProvider) GenerateCredentialsReturnsOnCall(i int, result1 *providers.Credentials, result2 error) {
	fake.generateCredentialsMutex.Lock()
	defer fake.generateCredentialsMutex.Unlock()
	fake.GenerateCredentialsStub = nil
	if fake.generateCredentialsReturnsOnCall == nil {
		fake.generateCredentialsReturnsOnCall = make(map[int]struct {
			result1 *providers.Credentials
			result2 error
		})
	}
	fake.generateCredentialsReturnsOnCall[i] = struct {
		result1 *providers.Credentials
		result2 error
	}{result1, result2}
}

func (fake *FakeProvider) GetInstanceParameters(arg1 context.Context, arg2 string) (providers.InstanceParameters, error) {
	fake.getInstanceParametersMutex.Lock()
	ret, specificReturn := fake.getInstanceParametersReturnsOnCall[len(fake.getInstanceParametersArgsForCall)]
	fake.getInstanceParametersArgsForCall = append(fake.getInstanceParametersArgsForCall, struct {
		arg1 context.Context
		arg2 string
	}{arg1, arg2})
	stub := fake.GetInstanceParametersStub
	fakeReturns := fake.getInstanceParametersReturns
	fake.recordInvocation("GetInstanceParameters", []interface{}{arg1, arg2})
	fake.getInstanceParametersMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeProvider) GetInstanceParametersCallCount() int {
	fake.getInstanceParametersMutex.RLock()
	defer fake.getInstanceParametersMutex.RUnlock()
	return len(fake.getInstanceParametersArgsForCall)
}

func (fake *FakeProvider) GetInstanceParametersCalls(stub func(context.Context, string) (providers.InstanceParameters, error)) {
	fake.getInstanceParametersMutex.Lock()
	defer fake.getInstanceParametersMutex.Unlock()
	fake.GetInstanceParametersStub = stub
}

func (fake *FakeProvider) GetInstanceParametersArgsForCall(i int) (context.Context, string) {
	fake.getInstanceParametersMutex.RLock()
	defer fake.getInstanceParametersMutex.RUnlock()
	argsForCall := fake.getInstanceParametersArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeProvider) GetInstanceParametersReturns(result1 providers.InstanceParameters, result2 error) {
	fake.getInstanceParametersMutex.Lock()
	defer fake.getInstanceParametersMutex.Unlock()
	fake.GetInstanceParametersStub = nil
	fake.getInstanceParametersReturns = struct {
		result1 providers.InstanceParameters
		result2 error
	}{result1, result2}
}

func (fake *FakeProvider) GetInstanceParametersReturnsOnCall(i int, result1 providers.InstanceParameters, result2 error) {
	fake.getInstanceParametersMutex.Lock()
	defer fake.getInstanceParametersMutex.Unlock()
	fake.GetInstanceParametersStub = nil
	if fake.getInstanceParametersReturnsOnCall == nil {
		fake.getInstanceParametersReturnsOnCall = make(map[int]struct {
			result1 providers.InstanceParameters
			result2 error
		})
	}
	fake.getInstanceParametersReturnsOnCall[i] = struct {
		result1 providers.InstanceParameters
		result2 error
	}{result1, result2}
}

func (fake *FakeProvider) GetInstanceTags(arg1 context.Context, arg2 string) (map[string]string, error) {
	fake.getInstanceTagsMutex.Lock()
	ret, specificReturn := fake.getInstanceTagsReturnsOnCall[len(fake.getInstanceTagsArgsForCall)]
	fake.getInstanceTagsArgsForCall = append(fake.getInstanceTagsArgsForCall, struct {
		arg1 context.Context
		arg2 string
	}{arg1, arg2})
	stub := fake.GetInstanceTagsStub
	fakeReturns := fake.getInstanceTagsReturns
	fake.recordInvocation("GetInstanceTags", []interface{}{arg1, arg2})
	fake.getInstanceTagsMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeProvider) GetInstanceTagsCallCount() int {
	fake.getInstanceTagsMutex.RLock()
	defer fake.getInstanceTagsMutex.RUnlock()
	return len(fake.getInstanceTagsArgsForCall)
}

func (fake *FakeProvider) GetInstanceTagsCalls(stub func(context.Context, string) (map[string]string, error)) {
	fake.getInstanceTagsMutex.Lock()
	defer fake.getInstanceTagsMutex.Unlock()
	fake.GetInstanceTagsStub = stub
}

func (fake *FakeProvider) GetInstanceTagsArgsForCall(i int) (context.Context, string) {
	fake.getInstanceTagsMutex.RLock()
	defer fake.getInstanceTagsMutex.RUnlock()
	argsForCall := fake.getInstanceTagsArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeProvider) GetInstanceTagsReturns(result1 map[string]string, result2 error) {
	fake.getInstanceTagsMutex.Lock()
	defer fake.getInstanceTagsMutex.Unlock()
	fake.GetInstanceTagsStub = nil
	fake.getInstanceTagsReturns = struct {
		result1 map[string]string
		result2 error
	}{result1, result2}
}

func (fake *FakeProvider) GetInstanceTagsReturnsOnCall(i int, result1 map[string]string, result2 error) {
	fake.getInstanceTagsMutex.Lock()
	defer fake.getInstanceTagsMutex.Unlock()
	fake.GetInstanceTagsStub = nil
	if fake.getInstanceTagsReturnsOnCall == nil {
		fake.getInstanceTagsReturnsOnCall = make(map[int]struct {
			result1 map[string]string
			result2 error
		})
	}
	fake.getInstanceTagsReturnsOnCall[i] = struct {
		result1 map[string]string
		result2 error
	}{result1, result2}
}

func (fake *FakeProvider) ProgressState(arg1 context.Context, arg2 string, arg3 string, arg4 string) (providers.ServiceState, string, error) {
	fake.progressStateMutex.Lock()
	ret, specificReturn := fake.progressStateReturnsOnCall[len(fake.progressStateArgsForCall)]
	fake.progressStateArgsForCall = append(fake.progressStateArgsForCall, struct {
		arg1 context.Context
		arg2 string
		arg3 string
		arg4 string
	}{arg1, arg2, arg3, arg4})
	stub := fake.ProgressStateStub
	fakeReturns := fake.progressStateReturns
	fake.recordInvocation("ProgressState", []interface{}{arg1, arg2, arg3, arg4})
	fake.progressStateMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2, arg3, arg4)
	}
	if specificReturn {
		return ret.result1, ret.result2, ret.result3
	}
	return fakeReturns.result1, fakeReturns.result2, fakeReturns.result3
}

func (fake *FakeProvider) ProgressStateCallCount() int {
	fake.progressStateMutex.RLock()
	defer fake.progressStateMutex.RUnlock()
	return len(fake.progressStateArgsForCall)
}

func (fake *FakeProvider) ProgressStateCalls(stub func(context.Context, string, string, string) (providers.ServiceState, string, error)) {
	fake.progressStateMutex.Lock()
	defer fake.progressStateMutex.Unlock()
	fake.ProgressStateStub = stub
}

func (fake *FakeProvider) ProgressStateArgsForCall(i int) (context.Context, string, string, string) {
	fake.progressStateMutex.RLock()
	defer fake.progressStateMutex.RUnlock()
	argsForCall := fake.progressStateArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3, argsForCall.arg4
}

func (fake *FakeProvider) ProgressStateReturns(result1 providers.ServiceState, result2 string, result3 error) {
	fake.progressStateMutex.Lock()
	defer fake.progressStateMutex.Unlock()
	fake.ProgressStateStub = nil
	fake.progressStateReturns = struct {
		result1 providers.ServiceState
		result2 string
		result3 error
	}{result1, result2, result3}
}

func (fake *FakeProvider) ProgressStateReturnsOnCall(i int, result1 providers.ServiceState, result2 string, result3 error) {
	fake.progressStateMutex.Lock()
	defer fake.progressStateMutex.Unlock()
	fake.ProgressStateStub = nil
	if fake.progressStateReturnsOnCall == nil {
		fake.progressStateReturnsOnCall = make(map[int]struct {
			result1 providers.ServiceState
			result2 string
			result3 error
		})
	}
	fake.progressStateReturnsOnCall[i] = struct {
		result1 providers.ServiceState
		result2 string
		result3 error
	}{result1, result2, result3}
}

func (fake *FakeProvider) Provision(arg1 context.Context, arg2 string, arg3 providers.ProvisionParameters) error {
	fake.provisionMutex.Lock()
	ret, specificReturn := fake.provisionReturnsOnCall[len(fake.provisionArgsForCall)]
	fake.provisionArgsForCall = append(fake.provisionArgsForCall, struct {
		arg1 context.Context
		arg2 string
		arg3 providers.ProvisionParameters
	}{arg1, arg2, arg3})
	stub := fake.ProvisionStub
	fakeReturns := fake.provisionReturns
	fake.recordInvocation("Provision", []interface{}{arg1, arg2, arg3})
	fake.provisionMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2, arg3)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeProvider) ProvisionCallCount() int {
	fake.provisionMutex.RLock()
	defer fake.provisionMutex.RUnlock()
	return len(fake.provisionArgsForCall)
}

func (fake *FakeProvider) ProvisionCalls(stub func(context.Context, string, providers.ProvisionParameters) error) {
	fake.provisionMutex.Lock()
	defer fake.provisionMutex.Unlock()
	fake.ProvisionStub = stub
}

func (fake *FakeProvider) ProvisionArgsForCall(i int) (context.Context, string, providers.ProvisionParameters) {
	fake.provisionMutex.RLock()
	defer fake.provisionMutex.RUnlock()
	argsForCall := fake.provisionArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3
}

func (fake *FakeProvider) ProvisionReturns(result1 error) {
	fake.provisionMutex.Lock()
	defer fake.provisionMutex.Unlock()
	fake.ProvisionStub = nil
	fake.provisionReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeProvider) ProvisionReturnsOnCall(i int, result1 error) {
	fake.provisionMutex.Lock()
	defer fake.provisionMutex.Unlock()
	fake.ProvisionStub = nil
	if fake.provisionReturnsOnCall == nil {
		fake.provisionReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.provisionReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeProvider) RevokeCredentials(arg1 context.Context, arg2 string, arg3 string) error {
	fake.revokeCredentialsMutex.Lock()
	ret, specificReturn := fake.revokeCredentialsReturnsOnCall[len(fake.revokeCredentialsArgsForCall)]
	fake.revokeCredentialsArgsForCall = append(fake.revokeCredentialsArgsForCall, struct {
		arg1 context.Context
		arg2 string
		arg3 string
	}{arg1, arg2, arg3})
	stub := fake.RevokeCredentialsStub
	fakeReturns := fake.revokeCredentialsReturns
	fake.recordInvocation("RevokeCredentials", []interface{}{arg1, arg2, arg3})
	fake.revokeCredentialsMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2, arg3)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeProvider) RevokeCredentialsCallCount() int {
	fake.revokeCredentialsMutex.RLock()
	defer fake.revokeCredentialsMutex.RUnlock()
	return len(fake.revokeCredentialsArgsForCall)
}

func (fake *FakeProvider) RevokeCredentialsCalls(stub func(context.Context, string, string) error) {
	fake.revokeCredentialsMutex.Lock()
	defer fake.revokeCredentialsMutex.Unlock()
	fake.RevokeCredentialsStub = stub
}

func (fake *FakeProvider) RevokeCredentialsArgsForCall(i int) (context.Context, string, string) {
	fake.revokeCredentialsMutex.RLock()
	defer fake.revokeCredentialsMutex.RUnlock()
	argsForCall := fake.revokeCredentialsArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3
}

func (fake *FakeProvider) RevokeCredentialsReturns(result1 error) {
	fake.revokeCredentialsMutex.Lock()
	defer fake.revokeCredentialsMutex.Unlock()
	fake.RevokeCredentialsStub = nil
	fake.revokeCredentialsReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeProvider) RevokeCredentialsReturnsOnCall(i int, result1 error) {
	fake.revokeCredentialsMutex.Lock()
	defer fake.revokeCredentialsMutex.Unlock()
	fake.RevokeCredentialsStub = nil
	if fake.revokeCredentialsReturnsOnCall == nil {
		fake.revokeCredentialsReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.revokeCredentialsReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeProvider) StartFailoverTest(arg1 context.Context, arg2 string) (string, error) {
	fake.startFailoverTestMutex.Lock()
	ret, specificReturn := fake.startFailoverTestReturnsOnCall[len(fake.startFailoverTestArgsForCall)]
	fake.startFailoverTestArgsForCall = append(fake.startFailoverTestArgsForCall, struct {
		arg1 context.Context
		arg2 string
	}{arg1, arg2})
	stub := fake.StartFailoverTestStub
	fakeReturns := fake.startFailoverTestReturns
	fake.recordInvocation("StartFailoverTest", []interface{}{arg1, arg2})
	fake.startFailoverTestMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeProvider) StartFailoverTestCallCount() int {
	fake.startFailoverTestMutex.RLock()
	defer fake.startFailoverTestMutex.RUnlock()
	return len(fake.startFailoverTestArgsForCall)
}

func (fake *FakeProvider) StartFailoverTestCalls(stub func(context.Context, string) (string, error)) {
	fake.startFailoverTestMutex.Lock()
	defer fake.startFailoverTestMutex.Unlock()
	fake.StartFailoverTestStub = stub
}

func (fake *FakeProvider) StartFailoverTestArgsForCall(i int) (context.Context, string) {
	fake.startFailoverTestMutex.RLock()
	defer fake.startFailoverTestMutex.RUnlock()
	argsForCall := fake.startFailoverTestArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeProvider) StartFailoverTestReturns(result1 string, result2 error) {
	fake.startFailoverTestMutex.Lock()
	defer fake.startFailoverTestMutex.Unlock()
	fake.StartFailoverTestStub = nil
	fake.startFailoverTestReturns = struct {
		result1 string
		result2 error
	}{result1, result2}
}

func (fake *FakeProvider) StartFailoverTestReturnsOnCall(i int, result1 string, result2 error) {
	fake.startFailoverTestMutex.Lock()
	defer fake.startFailoverTestMutex.Unlock()
	fake.StartFailoverTestStub = nil
	if fake.startFailoverTestReturnsOnCall == nil {
		fake.startFailoverTestReturnsOnCall = make(map[int]struct {
			result1 string
			result2 error
		})
	}
	fake.startFailoverTestReturnsOnCall[i] = struct {
		result1 string
		result2 error
	}{result1, result2}
}

func (fake *FakeProvider) UpdateParamGroupParameters(arg1 context.Context, arg2 string, arg3 providers.UpdateParamGroupParameters) error {
	fake.updateParamGroupParametersMutex.Lock()
	ret, specificReturn := fake.updateParamGroupParametersReturnsOnCall[len(fake.updateParamGroupParametersArgsForCall)]
	fake.updateParamGroupParametersArgsForCall = append(fake.updateParamGroupParametersArgsForCall, struct {
		arg1 context.Context
		arg2 string
		arg3 providers.UpdateParamGroupParameters
	}{arg1, arg2, arg3})
	stub := fake.UpdateParamGroupParametersStub
	fakeReturns := fake.updateParamGroupParametersReturns
	fake.recordInvocation("UpdateParamGroupParameters", []interface{}{arg1, arg2, arg3})
	fake.updateParamGroupParametersMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2, arg3)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeProvider) UpdateParamGroupParametersCallCount() int {
	fake.updateParamGroupParametersMutex.RLock()
	defer fake.updateParamGroupParametersMutex.RUnlock()
	return len(fake.updateParamGroupParametersArgsForCall)
}

func (fake *FakeProvider) UpdateParamGroupParametersCalls(stub func(context.Context, string, providers.UpdateParamGroupParameters) error) {
	fake.updateParamGroupParametersMutex.Lock()
	defer fake.updateParamGroupParametersMutex.Unlock()
	fake.UpdateParamGroupParametersStub = stub
}

func (fake *FakeProvider) UpdateParamGroupParametersArgsForCall(i int) (context.Context, string, providers.UpdateParamGroupParameters) {
	fake.updateParamGroupParametersMutex.RLock()
	defer fake.updateParamGroupParametersMutex.RUnlock()
	argsForCall := fake.updateParamGroupParametersArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3
}

func (fake *FakeProvider) UpdateParamGroupParametersReturns(result1 error) {
	fake.updateParamGroupParametersMutex.Lock()
	defer fake.updateParamGroupParametersMutex.Unlock()
	fake.UpdateParamGroupParametersStub = nil
	fake.updateParamGroupParametersReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeProvider) UpdateParamGroupParametersReturnsOnCall(i int, result1 error) {
	fake.updateParamGroupParametersMutex.Lock()
	defer fake.updateParamGroupParametersMutex.Unlock()
	fake.UpdateParamGroupParametersStub = nil
	if fake.updateParamGroupParametersReturnsOnCall == nil {
		fake.updateParamGroupParametersReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.updateParamGroupParametersReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeProvider) UpdateReplicationGroup(arg1 context.Context, arg2 string, arg3 providers.UpdateReplicationGroupParameters) error {
	fake.updateReplicationGroupMutex.Lock()
	ret, specificReturn := fake.updateReplicationGroupReturnsOnCall[len(fake.updateReplicationGroupArgsForCall)]
	fake.updateReplicationGroupArgsForCall = append(fake.updateReplicationGroupArgsForCall, struct {
		arg1 context.Context
		arg2 string
		arg3 providers.UpdateReplicationGroupParameters
	}{arg1, arg2, arg3})
	stub := fake.UpdateReplicationGroupStub
	fakeReturns := fake.updateReplicationGroupReturns
	fake.recordInvocation("UpdateReplicationGroup", []interface{}{arg1, arg2, arg3})
	fake.updateReplicationGroupMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2, arg3)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeProvider) UpdateReplicationGroupCallCount() int {
	fake.updateReplicationGroupMutex.RLock()
	defer fake.updateReplicationGroupMutex.RUnlock()
	return len(fake.updateReplicationGroupArgsForCall)
}

func (fake *FakeProvider) UpdateReplicationGroupCalls(stub func(context.Context, string, providers.UpdateReplicationGroupParameters) error) {
	fake.updateReplicationGroupMutex.Lock()
	defer fake.updateReplicationGroupMutex.Unlock()
	fake.UpdateReplicationGroupStub = stub
}

func (fake *FakeProvider) UpdateReplicationGroupArgsForCall(i int) (context.Context, string, providers.UpdateReplicationGroupParameters) {
	fake.updateReplicationGroupMutex.RLock()
	defer fake.updateReplicationGroupMutex.RUnlock()
	argsForCall := fake.updateReplicationGroupArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3
}

func (fake *FakeProvider) UpdateReplicationGroupReturns(result1 error) {
	fake.updateReplicationGroupMutex.Lock()
	defer fake.updateReplicationGroupMutex.Unlock()
	fake.UpdateReplicationGroupStub = nil
	fake.updateReplicationGroupReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeProvider) UpdateReplicationGroupReturnsOnCall(i int, result1 error) {
	fake.updateReplicationGroupMutex.Lock()
	defer fake.updateReplicationGroupMutex.Unlock()
	fake.UpdateReplicationGroupStub = nil
	if fake.updateReplicationGroupReturnsOnCall == nil {
		fake.updateReplicationGroupReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.updateReplicationGroupReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeProvider) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.deleteCacheParameterGroupMutex.RLock()
	defer fake.deleteCacheParameterGroupMutex.RUnlock()
	fake.deprovisionMutex.RLock()
	defer fake.deprovisionMutex.RUnlock()
	fake.findSnapshotsMutex.RLock()
	defer fake.findSnapshotsMutex.RUnlock()
	fake.generateCredentialsMutex.RLock()
	defer fake.generateCredentialsMutex.RUnlock()
	fake.getInstanceParametersMutex.RLock()
	defer fake.getInstanceParametersMutex.RUnlock()
	fake.getInstanceTagsMutex.RLock()
	defer fake.getInstanceTagsMutex.RUnlock()
	fake.progressStateMutex.RLock()
	defer fake.progressStateMutex.RUnlock()
	fake.provisionMutex.RLock()
	defer fake.provisionMutex.RUnlock()
	fake.revokeCredentialsMutex.RLock()
	defer fake.revokeCredentialsMutex.RUnlock()
	fake.startFailoverTestMutex.RLock()
	defer fake.startFailoverTestMutex.RUnlock()
	fake.updateParamGroupParametersMutex.RLock()
	defer fake.updateParamGroupParametersMutex.RUnlock()
	fake.updateReplicationGroupMutex.RLock()
	defer fake.updateReplicationGroupMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeProvider) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ providers.Provider = new(FakeProvider)
