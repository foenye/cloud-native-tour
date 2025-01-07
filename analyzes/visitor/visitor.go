package main

import "fmt"

type VisitorFn func() error
type Visitor interface {
	Visit(VisitorFn) error
}

var _ Visitor = VisitorImplementation{}

type VisitorImplementation struct {
}

func (tor VisitorImplementation) Visit(fn VisitorFn) error {
	fmt.Println("In VisitorImplementation before fn")
	_ = fn()
	fmt.Println("In VisitorImplementation after fn")
	return nil
}

var _ Visitor = VisitorList{}

type VisitorList []Visitor

func (visitors VisitorList) Visit(fn VisitorFn) error {
	for i := range visitors {
		if err := visitors[i].Visit(func() error {
			fmt.Println("In VisitorList before fn")
			_ = fn()
			fmt.Println("In VisitorList after fn")
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

var _ Visitor = VisitorWrapper{}

type VisitorWrapper struct {
	visitor Visitor
}

func (wrapper VisitorWrapper) Visit(fn VisitorFn) error {
	_ = wrapper.visitor.Visit(func() error {
		fmt.Println("In VisitorWrapper before fn")
		_ = fn()
		fmt.Println("In VisitorWrapper after fn")
		return nil
	})
	return nil
}

var _ Visitor = VisitorAggregator{}

type VisitorAggregator struct {
	visitor Visitor
}

func (aggregator VisitorAggregator) Visit(fn VisitorFn) error {
	_ = aggregator.visitor.Visit(func() error {
		fmt.Println("In VisitorAggregator before fn")
		_ = fn()
		fmt.Println("In VisitorAggregator after fn")
		return nil
	})
	return nil
}

func main() {
	var visitor Visitor
	var visitors []Visitor

	visitor = VisitorImplementation{}
	visitors = append(visitors, visitor)
	visitor = VisitorWrapper{VisitorList(visitors)}
	visitor = VisitorAggregator{visitor}
	_ = visitor.Visit(func() error {
		fmt.Println("In VisitFn")
		return nil
	})
}
