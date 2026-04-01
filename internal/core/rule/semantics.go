package rule

import (
	"encoding/json"
	"sort"
	"strings"
	"time"
)

type ShiftRelations struct {
	SubjectShiftIDs []string
	ObjectShiftIDs  []string
	TargetShiftIDs  []string
}

type ExclusiveSemantics struct {
	PrimaryShiftIDs []string
	RelatedShiftIDs []string
	TimeOffsetDays  int
	Symmetric       bool
}

type SourceSemantics struct {
	TargetShiftIDs []string
	SourceShiftIDs []string
	TimeOffsetDays int
}

type RequiredTogetherSemantics struct {
	TargetShiftIDs   []string
	RequiredShiftIDs []string
	TimeOffsetDays   int
}

type OrderSemantics struct {
	TargetShiftIDs []string
	BeforeShiftIDs []string
	TimeOffsetDays int
}

func (r Rule) AppliesToEmployee(employeeID string, employeeGroupIDs map[string]bool) bool {
	if strings.TrimSpace(employeeID) == "" || len(r.ApplyScopes) == 0 {
		return true
	}

	hasIncludeScope := false
	included := false
	excluded := false

	for _, scope := range r.ApplyScopes {
		scopeID := ""
		if scope.ScopeID != nil {
			scopeID = strings.TrimSpace(*scope.ScopeID)
		}

		switch strings.TrimSpace(scope.ScopeType) {
		case ScopeTypeAll:
			included = true
		case ScopeTypeEmployee:
			hasIncludeScope = true
			if scopeID != "" && scopeID == employeeID {
				included = true
			}
		case ScopeTypeGroup:
			hasIncludeScope = true
			if scopeID != "" && employeeGroupIDs != nil && employeeGroupIDs[scopeID] {
				included = true
			}
		case ScopeTypeExcludeEmployee:
			if scopeID != "" && scopeID == employeeID {
				excluded = true
			}
		case ScopeTypeExcludeGroup:
			if scopeID != "" && employeeGroupIDs != nil && employeeGroupIDs[scopeID] {
				excluded = true
			}
		}
	}

	if excluded {
		return false
	}
	if hasIncludeScope {
		return included
	}
	return true
}

func CollectGroupScopeIDs(rules []Rule) []string {
	result := make([]string, 0)
	seen := make(map[string]struct{})
	for _, currentRule := range rules {
		for _, scope := range currentRule.ApplyScopes {
			scopeID := ""
			if scope.ScopeID != nil {
				scopeID = strings.TrimSpace(*scope.ScopeID)
			}
			if scopeID == "" {
				continue
			}
			switch strings.TrimSpace(scope.ScopeType) {
			case ScopeTypeGroup, ScopeTypeExcludeGroup:
				appendUniqueString(&result, seen, scopeID)
			}
		}
	}
	return result
}

func (r Rule) ShiftRelations() ShiftRelations {
	relations := ShiftRelations{}
	seenSubject := make(map[string]struct{})
	seenObject := make(map[string]struct{})
	seenTarget := make(map[string]struct{})

	for _, assoc := range r.Associations {
		assocType := strings.TrimSpace(assoc.AssociationType)
		if assocType == "" {
			assocType = strings.TrimSpace(assoc.TargetType)
		}
		if assocType != TargetTypeShift {
			continue
		}

		assocID := strings.TrimSpace(assoc.AssociationID)
		if assocID == "" {
			assocID = strings.TrimSpace(assoc.TargetID)
		}
		if assocID == "" {
			continue
		}

		switch strings.TrimSpace(assoc.Role) {
		case AssociationRoleSubject:
			appendUniqueString(&relations.SubjectShiftIDs, seenSubject, assocID)
		case AssociationRoleObject, AssociationRoleSource:
			appendUniqueString(&relations.ObjectShiftIDs, seenObject, assocID)
		case AssociationRoleTarget, "":
			appendUniqueString(&relations.TargetShiftIDs, seenTarget, assocID)
		default:
			appendUniqueString(&relations.TargetShiftIDs, seenTarget, assocID)
		}
	}

	return relations
}

func (r Rule) ExclusiveSemantics() ExclusiveSemantics {
	relations := r.ShiftRelations()
	semantics := ExclusiveSemantics{
		PrimaryShiftIDs: mergeUniqueStrings(relations.SubjectShiftIDs, relations.TargetShiftIDs),
		RelatedShiftIDs: mergeUniqueStrings(relations.ObjectShiftIDs, nil),
		TimeOffsetDays:  r.timeOffsetDays(),
	}

	if len(semantics.PrimaryShiftIDs) > 0 && len(semantics.RelatedShiftIDs) > 0 {
		return semantics
	}
	if len(semantics.PrimaryShiftIDs) > 0 {
		semantics.Symmetric = true
		return semantics
	}

	var cfg ExclusiveShiftsConfig
	if err := json.Unmarshal(r.Config, &cfg); err == nil && len(cfg.ShiftIDs) > 0 {
		semantics.PrimaryShiftIDs = uniqueStrings(cfg.ShiftIDs)
		semantics.RelatedShiftIDs = nil
		semantics.Symmetric = true
	}

	return semantics
}

func (r Rule) SourceSemantics() SourceSemantics {
	relations := r.ShiftRelations()
	semantics := SourceSemantics{
		TargetShiftIDs: mergeUniqueStrings(relations.SubjectShiftIDs, relations.TargetShiftIDs),
		SourceShiftIDs: mergeUniqueStrings(relations.ObjectShiftIDs, nil),
		TimeOffsetDays: r.timeOffsetDays(),
	}

	if len(semantics.TargetShiftIDs) > 0 && len(semantics.SourceShiftIDs) > 0 {
		return semantics
	}

	var cfg StaffSourceConfig
	if err := json.Unmarshal(r.Config, &cfg); err == nil {
		if len(semantics.TargetShiftIDs) == 0 && strings.TrimSpace(cfg.TargetShiftID) != "" {
			semantics.TargetShiftIDs = []string{strings.TrimSpace(cfg.TargetShiftID)}
		}
		if len(semantics.SourceShiftIDs) == 0 && strings.TrimSpace(cfg.SourceShiftID) != "" {
			semantics.SourceShiftIDs = []string{strings.TrimSpace(cfg.SourceShiftID)}
		}
	}

	return semantics
}

func (r Rule) RequiredTogetherSemantics() RequiredTogetherSemantics {
	relations := r.ShiftRelations()
	semantics := RequiredTogetherSemantics{
		TargetShiftIDs:   mergeUniqueStrings(relations.SubjectShiftIDs, relations.TargetShiftIDs),
		RequiredShiftIDs: mergeUniqueStrings(relations.ObjectShiftIDs, nil),
		TimeOffsetDays:   r.timeOffsetDays(),
	}

	if len(semantics.TargetShiftIDs) > 0 && len(semantics.RequiredShiftIDs) > 0 {
		return semantics
	}

	var cfg RequiredTogetherConfig
	if err := json.Unmarshal(r.Config, &cfg); err == nil {
		if len(semantics.TargetShiftIDs) == 0 && strings.TrimSpace(cfg.ShiftID) != "" {
			semantics.TargetShiftIDs = []string{strings.TrimSpace(cfg.ShiftID)}
		}
	}

	return semantics
}

func (r Rule) OrderSemantics() OrderSemantics {
	relations := r.ShiftRelations()
	semantics := OrderSemantics{
		TargetShiftIDs: mergeUniqueStrings(relations.SubjectShiftIDs, relations.TargetShiftIDs),
		BeforeShiftIDs: mergeUniqueStrings(relations.ObjectShiftIDs, nil),
		TimeOffsetDays: r.timeOffsetDays(),
	}

	if len(semantics.TargetShiftIDs) > 0 && len(semantics.BeforeShiftIDs) > 0 {
		return semantics
	}

	var cfg ExecutionOrderConfig
	if err := json.Unmarshal(r.Config, &cfg); err == nil {
		if len(semantics.TargetShiftIDs) == 0 && strings.TrimSpace(cfg.AfterShiftID) != "" {
			semantics.TargetShiftIDs = []string{strings.TrimSpace(cfg.AfterShiftID)}
		}
		if len(semantics.BeforeShiftIDs) == 0 && strings.TrimSpace(cfg.BeforeShiftID) != "" {
			semantics.BeforeShiftIDs = []string{strings.TrimSpace(cfg.BeforeShiftID)}
		}
	}

	return semantics
}

func (s ExclusiveSemantics) CounterpartShiftIDs(currentShiftID string) ([]string, int, bool) {
	currentShiftID = strings.TrimSpace(currentShiftID)
	if currentShiftID == "" {
		return nil, 0, false
	}

	if s.Symmetric {
		if !containsString(s.PrimaryShiftIDs, currentShiftID) {
			return nil, 0, false
		}
		return excludeString(s.PrimaryShiftIDs, currentShiftID), s.TimeOffsetDays, true
	}
	if containsString(s.PrimaryShiftIDs, currentShiftID) {
		return s.RelatedShiftIDs, s.TimeOffsetDays, true
	}
	if containsString(s.RelatedShiftIDs, currentShiftID) {
		return s.PrimaryShiftIDs, -s.TimeOffsetDays, true
	}

	return nil, 0, false
}

func (s SourceSemantics) SourceShiftIDsForTarget(targetShiftID string) ([]string, int, bool) {
	targetShiftID = strings.TrimSpace(targetShiftID)
	if targetShiftID == "" || !containsString(s.TargetShiftIDs, targetShiftID) || len(s.SourceShiftIDs) == 0 {
		return nil, 0, false
	}
	return s.SourceShiftIDs, s.TimeOffsetDays, true
}

func (s RequiredTogetherSemantics) RequiredShiftIDsForTarget(targetShiftID string) ([]string, int, bool) {
	targetShiftID = strings.TrimSpace(targetShiftID)
	if targetShiftID == "" || !containsString(s.TargetShiftIDs, targetShiftID) || len(s.RequiredShiftIDs) == 0 {
		return nil, 0, false
	}
	return s.RequiredShiftIDs, s.TimeOffsetDays, true
}

func DateStringWithOffset(date string, offsetDays int) string {
	if offsetDays == 0 {
		return date
	}
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	return t.AddDate(0, 0, offsetDays).Format("2006-01-02")
}

func OrderRulesForExecution(rules []Rule) []Rule {
	ordered := SortRulesByDependencies(rules)
	return ResolveRuleConflicts(ordered)
}

func SortRulesByDependencies(rules []Rule) []Rule {
	if len(rules) <= 1 {
		return rules
	}

	ruleMap := make(map[string]Rule, len(rules))
	orderIndex := make(map[string]int, len(rules))
	inDegree := make(map[string]int, len(rules))
	graph := make(map[string]map[string]struct{}, len(rules))

	for idx, currentRule := range rules {
		ruleMap[currentRule.ID] = currentRule
		orderIndex[currentRule.ID] = idx
		inDegree[currentRule.ID] = 0
	}

	for _, currentRule := range rules {
		for _, dep := range currentRule.Dependencies {
			fromID := strings.TrimSpace(dep.DependentOnRuleID)
			toID := strings.TrimSpace(dep.DependentRuleID)
			if fromID == "" || toID == "" || fromID == toID {
				continue
			}
			if _, ok := ruleMap[fromID]; !ok {
				continue
			}
			if _, ok := ruleMap[toID]; !ok {
				continue
			}
			if graph[fromID] == nil {
				graph[fromID] = make(map[string]struct{})
			}
			if _, exists := graph[fromID][toID]; exists {
				continue
			}
			graph[fromID][toID] = struct{}{}
			inDegree[toID]++
		}
	}

	queue := make([]string, 0, len(rules))
	for _, currentRule := range rules {
		if inDegree[currentRule.ID] == 0 {
			queue = append(queue, currentRule.ID)
		}
	}
	sortRuleIDs(queue, ruleMap, orderIndex)

	result := make([]Rule, 0, len(rules))
	for len(queue) > 0 {
		currentID := queue[0]
		queue = queue[1:]
		result = append(result, ruleMap[currentID])

		nextIDs := make([]string, 0, len(graph[currentID]))
		for nextID := range graph[currentID] {
			nextIDs = append(nextIDs, nextID)
		}
		sortRuleIDs(nextIDs, ruleMap, orderIndex)

		for _, nextID := range nextIDs {
			inDegree[nextID]--
			if inDegree[nextID] == 0 {
				queue = append(queue, nextID)
				sortRuleIDs(queue, ruleMap, orderIndex)
			}
		}
	}

	if len(result) != len(rules) {
		return rules
	}
	return result
}

func ResolveRuleConflicts(rules []Rule) []Rule {
	if len(rules) <= 1 {
		return rules
	}

	kept := make(map[string]Rule, len(rules))
	result := make([]Rule, 0, len(rules))
	for _, currentRule := range rules {
		shouldSkip := false
		for _, conflict := range currentRule.Conflicts {
			counterpartID := conflictCounterpartID(currentRule.ID, conflict)
			if counterpartID == "" {
				continue
			}
			counterpartRule, exists := kept[counterpartID]
			if exists && rulesMayConflictInExecutionContext(currentRule, counterpartRule) {
				shouldSkip = true
				break
			}
		}
		if shouldSkip {
			continue
		}
		kept[currentRule.ID] = currentRule
		result = append(result, currentRule)
	}
	return result
}

func (r Rule) timeOffsetDays() int {
	if r.TimeOffsetDays != nil {
		return *r.TimeOffsetDays
	}
	return 0
}

func appendUniqueString(dst *[]string, seen map[string]struct{}, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	if _, ok := seen[value]; ok {
		return
	}
	seen[value] = struct{}{}
	*dst = append(*dst, value)
}

func uniqueStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		appendUniqueString(&result, seen, value)
	}
	return result
}

func mergeUniqueStrings(left []string, right []string) []string {
	result := make([]string, 0, len(left)+len(right))
	seen := make(map[string]struct{}, len(left)+len(right))
	for _, value := range left {
		appendUniqueString(&result, seen, value)
	}
	for _, value := range right {
		appendUniqueString(&result, seen, value)
	}
	return result
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func excludeString(values []string, target string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != target {
			result = append(result, value)
		}
	}
	return result
}

func sortRuleIDs(ruleIDs []string, ruleMap map[string]Rule, orderIndex map[string]int) {
	sort.Slice(ruleIDs, func(i, j int) bool {
		left := ruleMap[ruleIDs[i]]
		right := ruleMap[ruleIDs[j]]
		if left.Priority != right.Priority {
			return left.Priority < right.Priority
		}
		return orderIndex[left.ID] < orderIndex[right.ID]
	})
}

func conflictCounterpartID(ruleID string, conflict RuleConflict) string {
	currentID := strings.TrimSpace(ruleID)
	if currentID == "" {
		return ""
	}
	ruleID1 := strings.TrimSpace(conflict.RuleID1)
	ruleID2 := strings.TrimSpace(conflict.RuleID2)
	if ruleID1 == currentID {
		return ruleID2
	}
	if ruleID2 == currentID {
		return ruleID1
	}
	return ""
}

func rulesMayConflictInExecutionContext(left Rule, right Rule) bool {
	if !rulesMayOverlapByShift(left, right) {
		return false
	}
	if !rulesMayOverlapByScope(left, right) {
		return false
	}
	if !rulesMayOverlapByTime(left, right) {
		return false
	}
	return true
}

func ShouldSkipRuleInConflictContext(current Rule, activeRules []Rule, employeeID string, employeeGroupIDs map[string]bool, shiftID string) bool {
	if !current.AppliesToEmployee(employeeID, employeeGroupIDs) || !current.AppliesToShiftContext(shiftID) {
		return false
	}
	for _, activeRule := range activeRules {
		if !activeRule.IsEnabled {
			continue
		}
		if !activeRule.AppliesToEmployee(employeeID, employeeGroupIDs) || !activeRule.AppliesToShiftContext(shiftID) {
			continue
		}
		if !rulesConflictPair(current, activeRule) {
			continue
		}
		if !rulesMayOverlapByTime(current, activeRule) {
			continue
		}
		return true
	}
	return false
}

func ActiveRulesForShiftContext(rules []Rule, employeeID string, employeeGroupIDs map[string]bool, shiftID string) []Rule {
	activeRules := make([]Rule, 0, len(rules))
	for _, current := range rules {
		if !current.IsEnabled {
			continue
		}
		if !current.AppliesToEmployee(employeeID, employeeGroupIDs) {
			continue
		}
		if !current.AppliesToShiftContext(shiftID) {
			continue
		}
		if ShouldSkipRuleInConflictContext(current, activeRules, employeeID, employeeGroupIDs, shiftID) {
			continue
		}
		activeRules = append(activeRules, current)
	}
	return activeRules
}

func (r Rule) AppliesToShiftContext(shiftID string) bool {
	shiftID = strings.TrimSpace(shiftID)
	if shiftID == "" {
		return true
	}

	switch strings.TrimSpace(r.SubType) {
	case SubTypeSource:
		semantics := r.SourceSemantics()
		if len(semantics.TargetShiftIDs) > 0 {
			return containsString(semantics.TargetShiftIDs, shiftID)
		}
	case SubTypeMust:
		semantics := r.RequiredTogetherSemantics()
		if len(semantics.TargetShiftIDs) > 0 {
			return containsString(semantics.TargetShiftIDs, shiftID)
		}
	case SubTypeForbid:
		explicit := explicitRuleShiftIDs(r)
		if len(explicit) > 0 {
			return containsString(explicit, shiftID)
		}
	case SubTypeLimit:
		var cfg MaxCountConfig
		if err := json.Unmarshal(r.Config, &cfg); err == nil && strings.TrimSpace(cfg.ShiftID) != "" {
			return strings.TrimSpace(cfg.ShiftID) == shiftID
		}
	case SubTypePrefer:
		var cfg PreferEmployeeConfig
		if err := json.Unmarshal(r.Config, &cfg); err == nil && strings.TrimSpace(cfg.ShiftID) != "" {
			return strings.TrimSpace(cfg.ShiftID) == shiftID
		}
	}

	explicit := explicitRuleShiftIDs(r)
	if len(explicit) == 0 {
		return true
	}
	return containsString(explicit, shiftID)
}

func rulesMayOverlapByShift(left Rule, right Rule) bool {
	leftShiftIDs := explicitRuleShiftIDs(left)
	rightShiftIDs := explicitRuleShiftIDs(right)
	if len(leftShiftIDs) == 0 || len(rightShiftIDs) == 0 {
		return true
	}
	return sameStringSet(leftShiftIDs, rightShiftIDs)
}

func rulesMayOverlapByScope(left Rule, right Rule) bool {
	leftScope := explicitConflictScope(left)
	rightScope := explicitConflictScope(right)
	if leftScope.kind == conflictScopeUnknown || rightScope.kind == conflictScopeUnknown {
		return false
	}
	if leftScope.kind == conflictScopeGlobal && rightScope.kind == conflictScopeGlobal {
		return true
	}
	if leftScope.kind != rightScope.kind {
		return false
	}
	return sameStringSet(leftScope.ids, rightScope.ids)
}

func rulesMayOverlapByTime(left Rule, right Rule) bool {
	return normalizedTimeScope(left.TimeScope) == normalizedTimeScope(right.TimeScope) && left.timeOffsetDays() == right.timeOffsetDays()
}

func explicitRuleShiftIDs(currentRule Rule) []string {
	shiftIDs := currentRule.ShiftRelations()
	result := mergeUniqueStrings(shiftIDs.SubjectShiftIDs, shiftIDs.ObjectShiftIDs)
	result = mergeUniqueStrings(result, shiftIDs.TargetShiftIDs)

	exclusive := currentRule.ExclusiveSemantics()
	result = mergeUniqueStrings(result, exclusive.PrimaryShiftIDs)
	result = mergeUniqueStrings(result, exclusive.RelatedShiftIDs)

	source := currentRule.SourceSemantics()
	result = mergeUniqueStrings(result, source.TargetShiftIDs)
	result = mergeUniqueStrings(result, source.SourceShiftIDs)

	required := currentRule.RequiredTogetherSemantics()
	result = mergeUniqueStrings(result, required.TargetShiftIDs)
	result = mergeUniqueStrings(result, required.RequiredShiftIDs)

	var prefer PreferEmployeeConfig
	if err := json.Unmarshal(currentRule.Config, &prefer); err == nil && strings.TrimSpace(prefer.ShiftID) != "" {
		result = mergeUniqueStrings(result, []string{strings.TrimSpace(prefer.ShiftID)})
	}

	order := currentRule.OrderSemantics()
	result = mergeUniqueStrings(result, order.TargetShiftIDs)
	result = mergeUniqueStrings(result, order.BeforeShiftIDs)

	return result
}

type conflictScope struct {
	kind string
	ids  []string
}

const (
	conflictScopeUnknown  = "unknown"
	conflictScopeGlobal   = "global"
	conflictScopeEmployee = "employee"
	conflictScopeGroup    = "group"
)

func explicitConflictScope(currentRule Rule) conflictScope {
	if len(currentRule.ApplyScopes) == 0 {
		return conflictScope{kind: conflictScopeGlobal}
	}

	kind := ""
	ids := make([]string, 0, len(currentRule.ApplyScopes))
	seen := make(map[string]struct{}, len(currentRule.ApplyScopes))
	for _, scope := range currentRule.ApplyScopes {
		scopeType := strings.TrimSpace(scope.ScopeType)
		switch scopeType {
		case "", ScopeTypeAll:
			if kind != "" {
				return conflictScope{kind: conflictScopeUnknown}
			}
			kind = conflictScopeGlobal
		case ScopeTypeEmployee:
			if scope.ScopeID == nil {
				return conflictScope{kind: conflictScopeUnknown}
			}
			if kind == "" {
				kind = conflictScopeEmployee
			}
			if kind != conflictScopeEmployee {
				return conflictScope{kind: conflictScopeUnknown}
			}
			appendUniqueString(&ids, seen, *scope.ScopeID)
		case ScopeTypeGroup:
			if scope.ScopeID == nil {
				return conflictScope{kind: conflictScopeUnknown}
			}
			if kind == "" {
				kind = conflictScopeGroup
			}
			if kind != conflictScopeGroup {
				return conflictScope{kind: conflictScopeUnknown}
			}
			appendUniqueString(&ids, seen, *scope.ScopeID)
		default:
			return conflictScope{kind: conflictScopeUnknown}
		}
	}
	if kind == "" {
		return conflictScope{kind: conflictScopeUnknown}
	}
	if kind == conflictScopeGlobal {
		return conflictScope{kind: conflictScopeGlobal}
	}
	if len(ids) == 0 {
		return conflictScope{kind: conflictScopeUnknown}
	}
	return conflictScope{kind: kind, ids: ids}
}

func normalizedTimeScope(scope string) string {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return TimeScopeSameDay
	}
	return scope
}

func sameStringSet(left []string, right []string) bool {
	left = uniqueStrings(left)
	right = uniqueStrings(right)
	if len(left) != len(right) {
		return false
	}
	for _, leftValue := range left {
		if !containsString(right, leftValue) {
			return false
		}
	}
	return true
}

func rulesConflictPair(left Rule, right Rule) bool {
	return ruleHasConflictWith(left, right.ID) || ruleHasConflictWith(right, left.ID)
}

func ruleHasConflictWith(current Rule, counterpartID string) bool {
	for _, conflict := range current.Conflicts {
		if conflictCounterpartID(current.ID, conflict) == counterpartID {
			return true
		}
	}
	return false
}
