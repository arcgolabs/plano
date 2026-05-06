package compiler

func isArithmeticOp(op string) bool {
	return op == "+" || op == "-" || op == "*" || op == "/"
}

func isEqualityOp(op string) bool {
	return op == "==" || op == "!="
}

func isLogicalOp(op string) bool {
	return op == "&&" || op == "||"
}

func isMembershipOp(op string) bool {
	return op == "in"
}

func isComparisonOp(op string) bool {
	return op == ">" || op == ">=" || op == "<" || op == "<="
}
