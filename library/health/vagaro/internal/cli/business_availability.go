// Copyright 2026 Trevin Chow and contributors. Licensed under Apache-2.0. See LICENSE.
// Hand-authored Phase 3 foundation: next-available summary built on the
// getavailablemultiappointments slots primitive.

package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/mvanhorn/printing-press-library/library/health/vagaro/internal/vagaro"
	"github.com/spf13/cobra"
)

// availabilitySummary is the compact next-available view for a business.
type availabilitySummary struct {
	Slug        string             `json:"slug"`
	BusinessID  string             `json:"business_id"`
	ServiceID   int64              `json:"service_id,omitempty"`
	ServiceName string             `json:"service,omitempty"`
	WeekOf      string             `json:"week_of"`
	TotalSlots  int                `json:"total_slots"`
	Groups      []vagaro.SlotGroup `json:"groups"`
	Note        string             `json:"note,omitempty"`
}

func newBusinessAvailabilityCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "availability <slug>",
		Short:   "Summarize a business's next-available slots for the current week",
		Example: "  vagaro-pp-cli business availability centralbarber",
		// pp:data-source live
		Annotations: map[string]string{"mcp:read-only": "true", "pp:data-source": "live", "pp:happy-args": "slug=centralbarber"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			slug := strings.Trim(strings.TrimSpace(args[0]), "/")
			if slug == "" {
				return usageErr(fmt.Errorf("slug is required\nUsage: %s <slug>", cmd.CommandPath()))
			}
			if dryRunOK(flags) {
				return nil
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()

			c := newVagaroClient(flags)
			businessID, err := resolveBusinessID(ctx, c, flags, slug)
			if err != nil {
				return classifyVagaroError(err, flags)
			}
			services, err := c.Services(ctx, businessID)
			if err != nil {
				return classifyVagaroError(err, flags)
			}
			summary := availabilitySummary{Slug: slug, BusinessID: businessID, Groups: []vagaro.SlotGroup{}}
			if len(services) == 0 {
				summary.Note = "business has no bookable services"
				return emitVagaro(cmd, flags, summary)
			}
			svc := services[0]
			summary.ServiceID = svc.ServiceID
			summary.ServiceName = svc.ServiceTitle

			appDate, err := vagaro.FormatAppDate(time.Now().Format("2006-01-02"))
			if err != nil {
				return err
			}
			summary.WeekOf = appDate
			groups, err := c.Availability(ctx, businessID, fmt.Sprintf("%d", svc.ServiceID), "", appDate)
			if err != nil {
				return classifyVagaroError(err, flags)
			}
			summary.Groups = groups
			for _, g := range groups {
				summary.TotalSlots += len(g.Times)
			}
			if summary.TotalSlots == 0 {
				summary.Note = "no availability found for the current week"
			}
			return emitVagaro(cmd, flags, summary)
		},
	}
	return cmd
}
