package lang

// CollectParseErrors recursively collects any ParseError nodes in the provided
// AST node and returns them.
func CollectParseErrors(n interface{}) []*ParseError {
	var errs []*ParseError
	collectParseErrors(n, &errs)
	return errs
}

func collectParseErrors(n interface{}, errs *[]*ParseError) {
	switch t := n.(type) {
	case *ParseError:
		*errs = append(*errs, t)
	case *List:
		for _, c := range *t {
			collectParseErrors(c, errs)
		}
	case *Map:
		for k, v := range *t {
			collectParseErrors(&k, errs)
			collectParseErrors(v, errs)
		}
	case *Call:
		collectParseErrors(t.Func, errs)
		collectParseErrors(t.Arg, errs)
	case *Function:
		collectParseErrors(t.Param, errs)
		collectParseErrors(t.Body, errs)
	case *Shovel:
		collectParseErrors(t.Left, errs)
		collectParseErrors(t.Right, errs)
	case *Let:
		collectParseErrors(t.Bindings, errs)
		collectParseErrors(t.Body, errs)
	}
}
