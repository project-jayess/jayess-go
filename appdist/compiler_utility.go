package appdist

import "jayess-go/binding"

func PlanCompilerUtility(executablePath string, outputDir string, bindingPlan binding.BuildPlan, targetName string) Plan {
	return PlanApplication(executablePath, outputDir, bindingPlan, targetName)
}
