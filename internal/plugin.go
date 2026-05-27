package internal

import (
	"fmt"

	"github.com/GoCodeAlone/workflow-plugin-ory-polis/internal/contracts"
	pb "github.com/GoCodeAlone/workflow/plugin/external/proto"
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

var Version = "0.0.0"

type ory_polisPlugin struct{}

func NewOryPolisPlugin() sdk.PluginProvider {
	return &ory_polisPlugin{}
}

func (p *ory_polisPlugin) Manifest() sdk.PluginManifest {
	return sdk.PluginManifest{
		Name:        "workflow-plugin-ory-polis",
		Version:     Version,
		Author:      "GoCodeAlone",
		Description: "Ory Polis enterprise SSO and Directory Sync provider plugin",
	}
}

func (p *ory_polisPlugin) ModuleTypes() []string {
	return []string{"ory.polis"}
}

func (p *ory_polisPlugin) TypedModuleTypes() []string {
	return p.ModuleTypes()
}

func (p *ory_polisPlugin) CreateModule(typeName, name string, config map[string]any) (sdk.ModuleInstance, error) {
	switch typeName {
	case "ory.polis":
		return newOryPolisModule(name, config)
	default:
		return nil, fmt.Errorf("ory_polis plugin: unknown module type %q", typeName)
	}
}

func (p *ory_polisPlugin) CreateTypedModule(typeName, name string, config *anypb.Any) (sdk.ModuleInstance, error) {
	if typeName != "ory.polis" {
		return nil, fmt.Errorf("ory_polis plugin: unknown typed module type %q", typeName)
	}
	factory := sdk.NewTypedModuleFactory(typeName, &contracts.ProviderConfig{}, func(name string, cfg *contracts.ProviderConfig) (sdk.ModuleInstance, error) {
		return newOryPolisModule(name, typedModuleConfig(cfg))
	})
	return factory.CreateTypedModule(typeName, name, config)
}

func (p *ory_polisPlugin) StepTypes() []string {
	return allStepTypes()
}

func (p *ory_polisPlugin) TypedStepTypes() []string {
	return p.StepTypes()
}

func (p *ory_polisPlugin) CreateStep(typeName, name string, config map[string]any) (sdk.StepInstance, error) {
	return createStep(typeName, name, config)
}

func (p *ory_polisPlugin) CreateTypedStep(typeName, name string, config *anypb.Any) (sdk.StepInstance, error) {
	if _, ok := stepRegistry[typeName]; !ok {
		return nil, fmt.Errorf("%w: step type %q", sdk.ErrTypedContractNotHandled, typeName)
	}
	if typeName == "step.ory_polis_auth_provider_describe" {
		return sdk.NewTypedStepFactory(typeName, &contracts.AuthProviderDescribeConfig{}, &contracts.AuthProviderDescribeInput{}, typedAuthProviderDescribe).CreateTypedStep(typeName, name, config)
	}
	return sdk.NewTypedStepFactory(typeName, &contracts.OryPolisStepConfig{}, &contracts.OryPolisStepInput{}, typedStepHandler(typeName)).CreateTypedStep(typeName, name, config)
}

func (p *ory_polisPlugin) ContractRegistry() *pb.ContractRegistry {
	return contractRegistry
}

var contractRegistry = &pb.ContractRegistry{
	FileDescriptorSet: &descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{
			protodesc.ToFileDescriptorProto(structpb.File_google_protobuf_struct_proto),
			protodesc.ToFileDescriptorProto(contracts.File_internal_contracts_ory_polis_proto),
		},
	},
	Contracts: contractDescriptors(),
}

func contractDescriptors() []*pb.ContractDescriptor {
	descriptors := []*pb.ContractDescriptor{
		moduleContract("ory.polis", "ProviderConfig"),
	}
	for _, stepType := range allStepTypes() {
		if stepType == "step.ory_polis_auth_provider_describe" {
			descriptors = append(descriptors, stepContract(stepType, "AuthProviderDescribeConfig", "AuthProviderDescribeInput", "AuthProviderDescribeOutput"))
			continue
		}
		descriptors = append(descriptors, stepContract(stepType, "OryPolisStepConfig", "OryPolisStepInput", "OryPolisStepOutput"))
	}
	return descriptors
}

func moduleContract(moduleType, configMessage string) *pb.ContractDescriptor {
	const pkg = "workflow.plugins.ory_polis.v1."
	return &pb.ContractDescriptor{
		Kind:          pb.ContractKind_CONTRACT_KIND_MODULE,
		ModuleType:    moduleType,
		ConfigMessage: pkg + configMessage,
		Mode:          pb.ContractMode_CONTRACT_MODE_STRICT_PROTO,
	}
}

func stepContract(stepType, configMessage, inputMessage, outputMessage string) *pb.ContractDescriptor {
	const pkg = "workflow.plugins.ory_polis.v1."
	return &pb.ContractDescriptor{
		Kind:          pb.ContractKind_CONTRACT_KIND_STEP,
		StepType:      stepType,
		ConfigMessage: pkg + configMessage,
		InputMessage:  pkg + inputMessage,
		OutputMessage: pkg + outputMessage,
		Mode:          pb.ContractMode_CONTRACT_MODE_STRICT_PROTO,
	}
}
