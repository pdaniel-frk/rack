package models

import (
	"fmt"
	"strings"
)

type CronJob struct {
	Name          string `yaml:"name"`
	Schedule      string `yaml:"schedule"`
	Command       string `yaml:"command"`
	ManifestEntry ManifestEntry
}

func NewCronJobFromLabel(key, value string) CronJob {
	keySlice := strings.Split(key, ".")
	name := keySlice[len(keySlice)-1]
	tokens := strings.Fields(value)
	cronjob := CronJob{
		Name:     name,
		Schedule: fmt.Sprintf("cron(%s *)", strings.Join(tokens[0:5], " ")),
		Command:  strings.Join(tokens[5:], " "),
	}
	return cronjob
}

func (cr *CronJob) AppName() string {
	return cr.ManifestEntry.app.Name
}

func (cr *CronJob) Process() string {
	return cr.ManifestEntry.Name
}

func (cr *CronJob) ShortName() string {
	return fmt.Sprintf("%s%s", strings.Title(cr.ManifestEntry.Name), strings.Title(cr.Name))
}

func (cr *CronJob) LongName() string {
	app := cr.ManifestEntry.app
	return fmt.Sprintf("%s-%s-%s", app.StackName(), cr.Process(), cr.Name)
}
