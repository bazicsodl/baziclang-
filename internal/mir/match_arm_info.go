package mir

type matchArmInfo interface {
	matchArmVariant() string
	matchArmGuard() Expr
}

type matchArmTargetInfo interface {
	matchArmTarget() string
}

type matchArmGuardSetter interface {
	setMatchArmGuard(Expr)
}

func (a MatchArm) matchArmVariant() string           { return a.Variant }
func (a MatchArm) matchArmGuard() Expr               { return a.Guard }
func (a MatchExprArm) matchArmVariant() string       { return a.Variant }
func (a MatchExprArm) matchArmGuard() Expr           { return a.Guard }
func (a MatchTerminatorArm) matchArmVariant() string { return a.Variant }
func (a MatchTerminatorArm) matchArmGuard() Expr     { return a.Guard }
func (a MatchTerminatorArm) matchArmTarget() string  { return a.Target }
func (a *MatchArm) setMatchArmGuard(guard Expr)      { a.Guard = guard }
func (a *MatchExprArm) setMatchArmGuard(guard Expr)  { a.Guard = guard }
func (a *MatchTerminatorArm) setMatchArmGuard(guard Expr) {
	a.Guard = guard
}

func MatchArmVariant[T matchArmInfo](arm T) string {
	return arm.matchArmVariant()
}

func MatchArmGuard[T matchArmInfo](arm T) Expr {
	return arm.matchArmGuard()
}

func SetMatchArmGuard[T any](arm *T, guard Expr) bool {
	if arm == nil {
		return false
	}
	setter, ok := any(arm).(matchArmGuardSetter)
	if !ok {
		return false
	}
	setter.setMatchArmGuard(guard)
	return true
}

func MatchArmTarget[T matchArmTargetInfo](arm T) string {
	return arm.matchArmTarget()
}
