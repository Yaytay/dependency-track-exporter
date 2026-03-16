package render

import (
	"net/http"
	"sort"
	"strconv"
	"strings"

	"dependency-track-exporter/internal/client"
	"dependency-track-exporter/internal/snapshot"
)

type renderedProject struct {
	UUID       string
	Name       string
	Version    string
	Classifier string
	Tags       string
	Critical   int
	High       int
	Medium     int
	Low        int
	Unassigned int
}

func WriteMetrics(w http.ResponseWriter, snapshot snapshot.Snapshot) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	var b strings.Builder

	b.WriteString("# HELP dependency_track_up Whether the last Dependency-Track refresh succeeded.\n")
	b.WriteString("# TYPE dependency_track_up gauge\n")
	if snapshot.Up {
		b.WriteString("dependency_track_up 1\n")
	} else {
		b.WriteString("dependency_track_up 0\n")
	}

	b.WriteString("# HELP dependency_track_project_info Static project information.\n")
	b.WriteString("# TYPE dependency_track_project_info gauge\n")

	b.WriteString("# HELP dependency_track_project_vulnerabilities Number of project vulnerabilities by severity.\n")
	b.WriteString("# TYPE dependency_track_project_vulnerabilities gauge\n")

	projects := append([]renderedProject{}, flatten(snapshot)...)

	sort.Slice(projects, func(i, j int) bool {
		if projects[i].Name != projects[j].Name {
			return projects[i].Name < projects[j].Name
		}
		if projects[i].Version != projects[j].Version {
			return projects[i].Version < projects[j].Version
		}
		return projects[i].UUID < projects[j].UUID
	})

	for _, p := range projects {
		b.WriteString("dependency_track_project_info{")
		writeLabel(&b, "uuid", p.UUID)
		b.WriteByte(',')
		writeLabel(&b, "name", p.Name)
		b.WriteByte(',')
		writeLabel(&b, "version", p.Version)
		b.WriteByte(',')
		writeLabel(&b, "classifier", p.Classifier)
		b.WriteByte(',')
		writeLabel(&b, "tags", p.Tags)
		b.WriteString("} 1\n")

		writeVulnMetric(&b, p, "critical", p.Critical)
		writeVulnMetric(&b, p, "high", p.High)
		writeVulnMetric(&b, p, "medium", p.Medium)
		writeVulnMetric(&b, p, "low", p.Low)
		writeVulnMetric(&b, p, "unassigned", p.Unassigned)
	}

	_, _ = w.Write([]byte(b.String()))
}

func flatten(snapshot snapshot.Snapshot) []renderedProject {
	out := make([]renderedProject, 0, len(snapshot.Projects))

	for _, p := range snapshot.Projects {
		out = append(out, renderedProject{
			UUID:       p.Project.UUID,
			Name:       p.Project.Name,
			Version:    p.Project.Version,
			Classifier: p.Project.Classifier,
			Tags:       joinedProjectTags(p.Project.Tags),
			Critical:   p.Counts.Critical,
			High:       p.Counts.High,
			Medium:     p.Counts.Medium,
			Low:        p.Counts.Low,
			Unassigned: p.Counts.Unassigned,
		})
	}
	return out
}

func joinedProjectTags(tags []client.ProjectTag) string {
	if len(tags) == 0 {
		return ","
	}

	names := make([]string, 0, len(tags))
	for _, tag := range tags {
		if tag.Name == "" {
			continue
		}
		names = append(names, tag.Name)
	}

	if len(names) == 0 {
		return ","
	}

	sort.Strings(names)
	return "," + strings.Join(names, ",") + ","
}

func writeVulnMetric(b *strings.Builder, p renderedProject, severity string, value int) {
	b.WriteString("dependency_track_project_vulnerabilities{")
	writeLabel(b, "uuid", p.UUID)
	b.WriteByte(',')
	writeLabel(b, "name", p.Name)
	b.WriteByte(',')
	writeLabel(b, "version", p.Version)
	b.WriteByte(',')
	writeLabel(b, "severity", severity)
	b.WriteString("} ")
	b.WriteString(strconv.Itoa(value))
	b.WriteByte('\n')
}

func writeLabel(b *strings.Builder, key, value string) {
	b.WriteString(key)
	b.WriteString(`="`)
	b.WriteString(escapeLabelValue(value))
	b.WriteByte('"')
}

func escapeLabelValue(v string) string {
	v = strings.ReplaceAll(v, `\`, `\\`)
	v = strings.ReplaceAll(v, "\n", `\n`)
	v = strings.ReplaceAll(v, `"`, `\"`)
	return v
}
