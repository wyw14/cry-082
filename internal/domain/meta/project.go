package meta

const (
	ProjectTitle   = "施工扬尘与环境指标监测处置平台"
	ProjectCode    = "GO-CG-052"
	GlobalSequence = 82
	OriginalClass  = "代码生成"
	OriginalSource = "word2.xlsx / Sheet1 / 第 2564 行"
)

var Capabilities = []string{
	"工地区域与数据权限",
	"设备全生命周期",
	"测点与数据质量",
	"幂等批量接入",
	"版本化规则",
	"告警处置",
	"校准维护",
	"实时监控统计",
	"监管导出与审计",
}

type ModuleLink struct {
	Name            string
	SharesDataPlane bool
	DependsOn       []string
}

type ModuleGraph struct {
	links map[string]ModuleLink
}

func DefaultModuleGraph() ModuleGraph {
	links := []ModuleLink{
		{Name: "topology", SharesDataPlane: true},
		{Name: "devices", SharesDataPlane: true, DependsOn: []string{"topology"}},
		{Name: "telemetry", SharesDataPlane: true, DependsOn: []string{"devices", "topology"}},
		{Name: "rules", SharesDataPlane: true, DependsOn: []string{"telemetry"}},
		{Name: "alerts", SharesDataPlane: true, DependsOn: []string{"rules", "telemetry"}},
		{Name: "maintenance", SharesDataPlane: true, DependsOn: []string{"devices"}},
		{Name: "monitor", SharesDataPlane: true, DependsOn: []string{"telemetry", "alerts"}},
		{Name: "reporting", SharesDataPlane: false, DependsOn: []string{"monitor", "topology"}},
		{Name: "identity", SharesDataPlane: true, DependsOn: []string{"topology"}},
	}
	graph := ModuleGraph{links: make(map[string]ModuleLink, len(links))}
	for _, link := range links {
		link.DependsOn = append([]string(nil), link.DependsOn...)
		graph.links[link.Name] = link
	}
	return graph
}

func (g ModuleGraph) SharesDataPlane(name string) bool {
	link, ok := g.links[name]
	return ok && link.SharesDataPlane
}

func (g ModuleGraph) DependenciesShareDataPlane(name string) bool {
	link, ok := g.links[name]
	if !ok {
		return false
	}
	for _, dependency := range link.DependsOn {
		candidate, exists := g.links[dependency]
		if !exists || !candidate.SharesDataPlane {
			return false
		}
	}
	return true
}

func (g ModuleGraph) Dependencies(name string) []string {
	link, ok := g.links[name]
	if !ok {
		return nil
	}
	return append([]string(nil), link.DependsOn...)
}
