package main

import (
	"fmt"
	"github.com/aqilarik/ruletrace/ruletrace"
)

func main() {
	// keep using expr-lang expression
	input := `user.Group in ["admin", "moderator"] || user.Id == comment.UserId || len(tweets) > 1 || user.Name == "John Doe"`

	env := map[string]interface{}{
		"user":    map[string]interface{}{"Group": "admin", "Id": 1, "Name": "John Doe"},
		"comment": map[string]interface{}{"UserId": 1},
		"tweets":  []string{"t1", "t2", "t3"},
	}

	// Build specs using fingerprints of canonical atom expressions.
	// In production, you’d generate these from ListAtoms(input) and store them.
	specs := map[string]ruletrace.ConditionSpec{
		ruletrace.Fingerprint(`user.Group in ["admin", "moderator"]`): {
			ID:          "c_group",
			ReasonTrue:  "GROUP_ALLOWED",
			ReasonFalse: "GROUP_NOT_ALLOWED",
		},
		ruletrace.Fingerprint(`user.Id == comment.UserId`): {
			ID:          "c_owner",
			ReasonTrue:  "IS_OWNER",
			ReasonFalse: "NOT_OWNER",
		},
		ruletrace.Fingerprint(`len(tweets) > 1`): {
			ID:          "c_tweets",
			ReasonTrue:  "HAS_TWEETS",
			ReasonFalse: "NO_TWEETS",
		},
		ruletrace.Fingerprint(`user.Name == "John Doe"`): {
			ID:          "c_name",
			ReasonTrue:  "NAME_MATCH",
			ReasonFalse: "NAME_MISMATCH",
		},
	}

	err := ruletrace.ValidateSpecs(specs)
	if err != nil {
		panic(err)
	}

	tracer := ruletrace.New(
		env,
		ruletrace.WithMode(ruletrace.TraceAtomic),
		ruletrace.WithShortCircuit(true),
		ruletrace.WithCond(true),
	)

	res := tracer.Trace(input, specs)

	fmt.Println("MODE:", res.Mode.String())
	fmt.Println("SOURCE:")
	fmt.Println(res.Source)
	fmt.Println("\nFINAL:", res.Final)

	fmt.Println("\nCHUNKS:")
	for _, c := range res.Chunks {
		fmt.Printf("- id=%q fp=%s expr=%s val=%v skipped=%v reason=%q err=%q\n",
			c.ID, c.Fingerprint, c.Expr, c.Value, c.Skipped, c.Reason, c.Error)
	}

	fmt.Println("\n====================================================================================\n")

	// if your tracer enabled Cond registration (EnableCond: true or equivalent) so expr can compile Cond(...) as a function.
	input2 := `len(tweets)+len(tweets)`
	res2 := tracer.Trace(input2, nil)
	fmt.Println("MODE:", res2.Mode.String())
	fmt.Println("SOURCE:")
	fmt.Println(res2.Source)
	fmt.Println("\nFINAL:", res2.Final)

	fmt.Println("\nCHUNKS:")
	for _, c := range res2.Chunks {
		fmt.Printf("- id=%q fp=%s expr=%s val=%v skipped=%v reason=%q err=%q\n",
			c.ID, c.Fingerprint, c.Expr, c.Value, c.Skipped, c.Reason, c.Error)
	}

}
