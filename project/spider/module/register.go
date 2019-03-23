package module

import (
	"errors"
	"fmt"
	"spider/exceptions"
	"sync"
)

var ErrNotFoundModuleInstance = errors.New("not found module instance error")

type register struct {
	moduleTypeMap map[Type]map[MID]Module
	lock          sync.RWMutex
}

func (r *register) Register(module Module) (bool, error) {

	if module == nil {
		return false, errors.New("nil module instance")
	}

	mid := module.Id()

	parts, err := SplitMid(mid)
	if err != nil {
		return false, err
	}

	moduleType := legalLetterTypeMap[parts[0]]

	if !CheckType(Type(moduleType), module) {
		err := fmt.Sprintf("incorrect module type: %s", moduleType)
		return false, errors.New(err)
	}

	r.lock.Lock()
	defer r.lock.Unlock()

	modules := r.moduleTypeMap[Type(moduleType)]

	if modules == nil {
		modules = map[MID]Module{}
	}

	if _, ok := modules[mid]; ok {
		return false, nil
	}
	modules[mid] = module
	r.moduleTypeMap[Type(moduleType)] = modules
	return true, nil
}

func (r *register) Unregister(mid MID) (bool, error) {
	parts, err := SplitMid(mid)
	if err != nil {
		return false, err
	}

	moduleType := legalLetterTypeMap[string(parts[0])]
	var deleted bool

	r.lock.Lock()
	defer r.lock.Unlock()

	if mod, ok := r.moduleTypeMap[moduleType]; ok {
		if _, ok = mod[mid]; ok {
			delete(mod, mid)
			deleted = true
		}
	}
	return deleted, nil
}

func (r *register) Get(moduleType Type) (Module, error) {
	modules, err := r.GetAllByType(moduleType)
	if err != nil {
		return nil, err
	}

	minScore := uint64(0)
	var selectModule Module
	for _, module := range modules {
		SetScore(&module)
		if err != nil {
			return nil, err
		}
		score := module.Score()
		if minScore == 0 || score < minScore {
			selectModule = module
			minScore = score
		}
	}
	return selectModule, nil
}

func (r *register) GetAllByType(moduleType Type) (map[MID]Module, error) {
	if !LegalType(moduleType) {
		err := exceptions.NewIllegalParameterError(fmt.Sprintf("illegal module type: %s", moduleType))
		return nil, err
	}

	r.lock.Lock()
	defer r.lock.Unlock()

	module := r.moduleTypeMap[moduleType]
	if len(module) == 0 {
		return nil, ErrNotFoundModuleInstance
	}

	result := map[MID]Module{}

	for mid, mod := range module {
		result[mid] = mod
	}

	return result, nil
}

func (r *register) GetAll() map[MID]Module {
	 result := map[MID]Module{}

	 r.lock.Lock()
	 defer r.lock.Unlock()

	 for _,mod := range r.moduleTypeMap{
	 	for mid,m := range mod{
	 		result[mid] = m
		}
	 }

	 return result
}

// 清除所有组件记录
func (r *register) Clear() {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.moduleTypeMap = map[Type]map[MID]Module{}
}

func NewRegister() Register {
	return &register{moduleTypeMap: map[Type]map[MID]Module{}}
}
