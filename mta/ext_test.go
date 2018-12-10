package mta

import (
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"
)

var _ = Describe("Ext", func() {
	modules := []*ModuleExt{
		{
			Name: "backend",
			BuildParams: map[string]interface{}{
				"ignore": []interface{}{".gitignore", "*.mtar"},
				"e":      "b",
			},
			Provides: []Provides{
				{
					Name: "backend_task",
					Properties: map[string]interface{}{
						"url": "${default-url}/tasks",
					},
				},
			},
			Requires: []Requires{
				{
					Name: "scheduler_api",
					Properties: map[string]interface{}{
						"scheduler_url": "~{url}",
					},
				},
			},
		},
		{
			Name: "scheduler",
			Provides: []Provides{
				{
					Name: "scheduler_api",
					Properties: map[string]interface{}{
						"url": "${default-url}/api/v2",
					},
				},
			},
			Requires: []Requires{
				{
					Name: "backend_task",
					Properties: map[string]interface{}{
						"task_url": "~{url}",
					},
				},
			},
		},
	}
	schemaVersion := "3.2"
	ext := &EXT{
		SchemaVersion: &schemaVersion,
		ID:            "com.acme.scheduling.ext",
		Extends:       "com.acme.scheduling",
		Targets:       []interface{}{"DEV"},
		Modules:       modules,
		Resources: []*Resource{
			{
				Name:   "plugins",
				Active: false,
				Requires: []Requires{
					{
						Name: "scheduler_api",
						Parameters: map[string]interface{}{
							"par2": "value2",
						},
						Properties: map[string]interface{}{
							"prop2": "a-prop-value",
						},
					},
				},
				Parameters: map[string]interface{}{
					"filter": map[interface{}]interface{}{"type": "com.acme.plugin"},
				},
				Properties: map[string]interface{}{
					"plugin_name": "${name}",
					"plugin_url":  "${url}/sources",
				},
			},
		},
		ModuleTypes: []*ModuleTypes{
			{
				Name: "tomcat",
				Parameters: map[string]interface{}{
					"buildpack": "java_buildpack",
				},
			},
		},
		ResourceTypes: []*ResourceTypes{
			{
				Name: "postgresql",
				Parameters: map[string]interface{}{
					"service-plan": "v9.4-small",
				},
			},
		},
		Parameters: map[string]interface{}{
			"first_param":  "a value",
			"second_param": []interface{}{"has", "structure"},
		},
	}
	var _ = Describe("EXT tests", func() {

		var _ = Describe("Parsing", func() {
			It("Modules parsing - sanity", func() {
				extFile, _ := ioutil.ReadFile("./testdata/mtaExt.yaml")
				// Unmarshal file
				oExt := &EXT{}
				Ω(yaml.Unmarshal(extFile, oExt)).Should(Succeed())
				Ω(oExt.Modules).Should(HaveLen(2))
			})
		})
	})

	var _ = Describe("UnmarshalExt", func() {
		It("Sanity", func() {
			wd, err := os.Getwd()
			Ω(err).Should(Succeed())
			content, err := ioutil.ReadFile(filepath.Join(wd, "testdata", "mtaExt.yaml"))
			Ω(err).Should(Succeed())
			m, err := UnmarshalExt(content)
			Ω(err).Should(Succeed())
			Ω(*ext).Should(BeEquivalentTo(*m))
			Ω(len(m.Modules)).Should(Equal(2))
		})
		It("Invalid content", func() {
			_, err := UnmarshalExt([]byte("wrong mtaExt"))
			Ω(err).Should(HaveOccurred())
		})
	})

	var _ = Describe("extendMap", func() {
		var m1 map[string]interface{}
		var m2 map[string]interface{}
		var m3 map[string]interface{}
		var m4 map[string]interface{}

		BeforeEach(func() {
			m1 = make(map[string]interface{})
			m2 = make(map[string]interface{})
			m3 = make(map[string]interface{})
			m4 = nil
			m1["a"] = "aa"
			m1["b"] = "xx"
			m2["b"] = "bb"
			m3["c"] = "cc"
		})

		var _ = DescribeTable("Sanity", func(m *map[string]interface{}, e *map[string]interface{}, ln int, key string, value interface{}) {
			extendMap(m, e)
			Ω(len(*m)).Should(Equal(ln))

			if value != nil {
				Ω((*m)[key]).Should(Equal(value))
			} else {
				Ω((*m)[key]).Should(BeNil())
			}
		},
			Entry("overwrite", &m1, &m2, 2, "b", "bb"),
			Entry("add", &m1, &m3, 3, "c", "cc"),
			Entry("res equals ext", &m4, &m1, 2, "b", "xx"),
			Entry("nothing to add", &m1, &m4, 2, "b", "xx"),
			Entry("both nil", &m4, &m4, 0, "b", nil),
		)
	})

	var _ = Describe("MergeMtaAndExt", func() {
		It("Sanity", func() {
			moduleA := Module{
				Name: "modA",
				Properties: map[string]interface{}{
					"a": "aa",
					"b": "xx",
				},
			}
			moduleB := Module{
				Name: "modB",
				Properties: map[string]interface{}{
					"b": "yy",
				},
			}
			moduleAExt := ModuleExt{
				Name: "modA",
				Properties: map[string]interface{}{
					"a": "aa",
					"b": "bb",
				},
			}
			mta := MTA{
				Modules: []*Module{&moduleA, &moduleB},
			}
			mtaExt := EXT{
				Modules: []*ModuleExt{&moduleAExt},
			}
			Merge(&mta, &mtaExt)
			m, err := mta.GetModuleByName("modA")
			Ω(err).Should(Succeed())
			Ω(m.Properties["b"]).Should(Equal("bb"))
		})
	})

})
