// Package format renders package results as human tables or JSON.
package format

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/ramtinhoss/pkgr/internal/manager"
)

func HumanSearch(w io.Writer, pkgs []manager.Package) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tVERSION\tPM\tINSTALLED\tDESCRIPTION")
	for _, p := range pkgs {
		inst := "no"
		if p.Installed {
			inst = "yes"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", p.Name, p.Version, p.Manager, inst, p.Description)
	}
	return tw.Flush()
}

func HumanList(w io.Writer, pkgs []manager.Package) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tVERSION\tPM\tLATEST")
	for _, p := range pkgs {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", p.Name, p.Version, p.Manager, p.Latest)
	}
	return tw.Flush()
}

func HumanInfo(w io.Writer, p manager.Package) error {
	fmt.Fprintf(w, "Name:        %s\n", p.Name)
	fmt.Fprintf(w, "Manager:     %s\n", p.Manager)
	fmt.Fprintf(w, "Version:     %s\n", p.Version)
	if p.Latest != "" && p.Latest != p.Version {
		fmt.Fprintf(w, "Latest:      %s\n", p.Latest)
	}
	fmt.Fprintf(w, "Installed:   %v\n", p.Installed)
	if p.Homepage != "" {
		fmt.Fprintf(w, "Homepage:    %s\n", p.Homepage)
	}
	if p.Description != "" {
		fmt.Fprintf(w, "Description: %s\n", p.Description)
	}
	return nil
}
