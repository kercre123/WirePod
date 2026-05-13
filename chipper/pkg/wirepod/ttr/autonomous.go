package wirepod_ttr

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fforchino/vector-go-sdk/pkg/vector"
	"github.com/fforchino/vector-go-sdk/pkg/vectorpb"
	"github.com/kercre123/wire-pod/chipper/pkg/logger"
	"github.com/kercre123/wire-pod/chipper/pkg/vars"
	"github.com/sashabaranov/go-openai"
	// "io" // If CreateChatCompletionStream.Recv() returns io.EOF
)

type Command struct {
	Name string
	Args []string
}

// Placeholder for actual camera function
func getCameraImage(robot *vector.Vector, ctx context.Context) ([]byte, error) {
	logger.Println("Attempting to get camera image...")
	// This is a placeholder. Actual implementation depends on vector-go-sdk.
	// The SDK might have a helper like robot.Camera.GetFrame() or robot.GetLatestFrame()
	// Or it might involve starting a stream and taking one image from services like robot.Conn.EnableCameraFeed()
	// For now, log and return nil. This part will need actual SDK functions.
	logger.Println("GetCameraImage: Not implemented, returning nil data.")
	return nil, nil
}

func CreateAutonomousAIReq(imageData []byte, esn string, modelName string) openai.ChatCompletionRequest {
	systemPrompt := `You are Vector, an autonomous robot. Your goal is to explore your environment.
You may receive an image description (if available). Based on this, decide on a sequence of actions.
Output your actions as commands, one command per line. Each command must be in the format CMD_COMMAND_NAME(ARG1, ARG2, ...).
Available commands:
- CMD_DRIVE_WHEELS(left_wheel_speed_mmps, right_wheel_speed_mmps, duration_ms)
- CMD_TURN_IN_PLACE(angle_degrees, speed_degrees_per_second) // Positive angle for left turn
- CMD_SAY(text)
- CMD_WAIT(duration_ms)
- CMD_SET_LIFT_HEIGHT(height_0_to_1) // 0.0 is down, 1.0 is up
- CMD_SET_HEAD_ANGLE(angle_degrees) // e.g., -22 (down) to 45 (up)
- CMD_PLAY_ANIMATION(animation_name)
- CMD_STOP_AUTONOMY() // Command to stop autonomous mode

Example:
CMD_DRIVE_WHEELS(50.0, 50.0, 1000)
CMD_SAY(I am moving forward.)`

	// TODO: Add image data to prompt if available and LLM supports it.
	// This part needs to be implemented based on go-openai capabilities for multimodal input.
	// For now, image data is ignored.

	var nChat []openai.ChatCompletionMessage
	smsg := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: systemPrompt,
	}
	nChat = append(nChat, smsg)

	// Add remembered chat if enabled and desired for autonomous mode
	// if vars.APIConfig.Knowledge.SaveChat {
	//  rchat := GetChat(esn) // Assuming GetChat is accessible from kgsim.go
	//	nChat = append(nChat, rchat.Chats...)
	// }

	nChat = append(nChat, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		// Content: "Vector, what is your next action based on your current understanding of the environment?",
		// More sophisticated prompt might include current sensor data if available as text
		Content: "Analyze your surroundings and decide on the next action or sequence of actions.",
	})

	var modelToUse string
	if vars.APIConfig.Knowledge.Provider == "openai" {
		modelToUse = openai.GPT4oMini // Default, can be overridden
		if modelName != "" {
			modelToUse = modelName
		}
	} else if vars.APIConfig.Knowledge.Provider == "together" || vars.APIConfig.Knowledge.Provider == "custom" {
		modelToUse = vars.APIConfig.Knowledge.Model // From config
		if modelName != "" {                         // Allow override
			modelToUse = modelName
		}
		if modelToUse == "" && vars.APIConfig.Knowledge.Provider == "together" { // Default for together if empty
			modelToUse = "meta-llama/Llama-3-70b-chat-hf"
		}
	} else {
		// Fallback or error if no provider matches known types with model settings
		logger.Println("Warning: LLM provider for autonomous mode might not have a model configured, using default GPT-4o Mini if OpenAI key is present.")
		modelToUse = openai.GPT4oMini // A general fallback
	}
	logger.Println("Autonomous mode using LLM model: ", modelToUse)

	return openai.ChatCompletionRequest{
		Model:            modelToUse,
		MaxTokens:        300, // Adjusted for commands, potentially multiple commands
		Temperature:      vars.APIConfig.Knowledge.Temperature, // Use configured temp
		TopP:             vars.APIConfig.Knowledge.TopP,         // Use configured top_p
		FrequencyPenalty: 0,
		PresencePenalty:  0,
		Messages:         nChat,
		Stream:           false, // Non-streaming for simpler command batch execution
	}
}

func ParseLLMCommands(llmResponse string) ([]Command, error) {
	var commands []Command
	cmdRegex := regexp.MustCompile(`CMD_([A-Z_0-9]+)\s*\(([^)]*)\)`)
	emptyArgCmdRegex := regexp.MustCompile(`CMD_([A-Z_0-9]+)\s*\(\s*\)`)

	lines := strings.Split(llmResponse, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "CMD_") {
			continue
		}

		var cmdName string
		var argsStr string
		var argsList []string

		matches := cmdRegex.FindStringSubmatch(line)
		if len(matches) == 3 {
			cmdName = matches[1]
			argsStr = matches[2]
			if strings.TrimSpace(argsStr) != "" {
				// Split arguments, handling potential spaces around commas
				rawArgs := strings.Split(argsStr, ",")
				for _, arg := range rawArgs {
					trimmedArg := strings.TrimSpace(arg)
					// Basic handling for quoted strings in CMD_SAY, remove quotes
					if strings.HasPrefix(trimmedArg, "\"") && strings.HasSuffix(trimmedArg, "\"") {
						trimmedArg = strings.Trim(trimmedArg, "\"")
					} else if strings.HasPrefix(trimmedArg, "'") && strings.HasSuffix(trimmedArg, "'") {
						trimmedArg = strings.Trim(trimmedArg, "'")
					}
					argsList = append(argsList, trimmedArg)
				}
			}
			commands = append(commands, Command{Name: cmdName, Args: argsList})
		} else {
			// Check for commands with no arguments like CMD_STOP_AUTONOMY()
			emptyMatches := emptyArgCmdRegex.FindStringSubmatch(line)
			if len(emptyMatches) == 2 {
				cmdName = emptyMatches[1]
				commands = append(commands, Command{Name: cmdName, Args: []string{}})
			} else {
				logger.Println("Could not parse command: ", line)
			}
		}
	}
	// No error if no commands found, could be valid (LLM decides to wait or output text not as command)
	return commands, nil
}

func ExecuteParsedCommands(robot *vector.Vector, commands []Command, ctx context.Context, stopBehaviorCtrl chan bool, stopLoop chan bool) error {
	if len(commands) == 0 {
		// logger.Println("No commands to execute from LLM.")
		return nil // Not an error, LLM might have just chatted or decided to wait.
	}
	logger.Println("Executing parsed commands: ", commands)

	for _, cmd := range commands {
		select {
		case <-ctx.Done():
			logger.Println("Context cancelled during command execution.")
			return ctx.Err()
		case <-stopLoop:
			logger.Println("Stop loop signal received during command execution.")
			return fmt.Errorf("autonomous mode stopped during command execution")
		default:
			// continue execution
		}

		logger.Println("Executing command: ", cmd.Name, "with args: ", cmd.Args)
		currentCmdSuccessful := true
		switch cmd.Name {
		case "DRIVE_WHEELS":
			if len(cmd.Args) != 3 {
				logger.Println("DRIVE_WHEELS expects 3 arguments, got ", cmd.Args)
				currentCmdSuccessful = false
				continue
			}
			leftSpeed, errL := strconv.ParseFloat(cmd.Args[0], 32)
			rightSpeed, errR := strconv.ParseFloat(cmd.Args[1], 32)
			duration, errD := strconv.ParseInt(cmd.Args[2], 10, 32)
			if errL != nil || errR != nil || errD != nil {
				logger.Println(fmt.Sprintf("Error parsing DRIVE_WHEELS arguments: %v, %v, %v", errL, errR, errD))
				currentCmdSuccessful = false
				continue
			}
			logger.Println(fmt.Sprintf("SDK Call: robot.Conn.DriveWheels(LeftWheelMmps: %f, RightWheelMmps: %f, DurationMs: %d)", float32(leftSpeed), float32(rightSpeed), int32(duration)))
			// Assuming DriveWheels is blocking or we wait for its duration.
			// The SDK's DriveWheels might be non-blocking and require a separate wait or a different command structure.
			// For now, assume it blocks or its effect is short enough.
			_, err := robot.Conn.DriveWheels(ctx, &vectorpb.DriveWheelsRequest{LeftWheelMmps: float32(leftSpeed), RightWheelMmps: float32(rightSpeed), LiftHeightMm: 0, DurationMs: int32(duration)})
			if err != nil {
				logger.Println("Error executing DriveWheels: ", err)
				currentCmdSuccessful = false
			}
			if int32(duration) > 0 && currentCmdSuccessful { // Only sleep if duration was specified and command sent
				time.Sleep(time.Duration(duration) * time.Millisecond)
			}

		case "TURN_IN_PLACE":
			if len(cmd.Args) != 2 {
				logger.Println("TURN_IN_PLACE expects 2 arguments, got ", cmd.Args)
				currentCmdSuccessful = false
				continue
			}
			angle, errA := strconv.ParseFloat(cmd.Args[0], 32)
			speed, errS := strconv.ParseFloat(cmd.Args[1], 32)
			if errA != nil || errS != nil {
				logger.Println(fmt.Sprintf("Error parsing TURN_IN_PLACE arguments: %v, %v", errA, errS))
				currentCmdSuccessful = false
				continue
			}
			logger.Println(fmt.Sprintf("SDK Call: robot.Conn.TurnInPlace(AngleDeg: %f, SpeedDegps: %f)", float32(angle), float32(speed)))
			_, err := robot.Conn.TurnInPlace(ctx, &vectorpb.TurnInPlaceRequest{AngleDeg: float32(angle), SpeedDegps: float32(speed), IsAbsolute: 0}) // Assuming relative turn
			if err != nil {
				logger.Println("Error executing TurnInPlace: ", err)
				currentCmdSuccessful = false
			}
			// Estimate turn duration to wait, or SDK handles this.
			if speed != 0 && currentCmdSuccessful {
				turnDurationMs := (absFloat32(float32(angle)) / absFloat32(float32(speed))) * 1000
				time.Sleep(time.Duration(turnDurationMs) * time.Millisecond)
			}


		case "SAY":
			if len(cmd.Args) != 1 {
				logger.Println("SAY expects 1 argument, got ", cmd.Args)
				currentCmdSuccessful = false
				continue
			}
			textToSay := cmd.Args[0]
			logger.Println(fmt.Sprintf("SDK Call: SayText(robot, \"%s\")", textToSay))
			// Use the sayText from bcontrol.go as it handles animations and behavior control nuances for speech
			sayText(robot, textToSay) // This function is defined in kgsim.go (or bcontrol.go from user)

		case "WAIT":
			if len(cmd.Args) != 1 {
				logger.Println("WAIT expects 1 argument, got ", cmd.Args)
				currentCmdSuccessful = false
				continue
			}
			durationMs, err := strconv.ParseInt(cmd.Args[0], 10, 32)
			if err != nil {
				logger.Println(fmt.Sprintf("Error parsing WAIT arguments: %v", err))
				currentCmdSuccessful = false
				continue
			}
			logger.Println(fmt.Sprintf("Waiting for %d ms", durationMs))
			time.Sleep(time.Duration(durationMs) * time.Millisecond)

		case "SET_LIFT_HEIGHT":
			if len(cmd.Args) != 1 {
				logger.Println("SET_LIFT_HEIGHT expects 1 argument, got ", cmd.Args)
				currentCmdSuccessful = false
				continue
			}
			heightRatio, err := strconv.ParseFloat(cmd.Args[0], 32)
			if err != nil {
				logger.Println(fmt.Sprintf("Error parsing SET_LIFT_HEIGHT arguments: %v", err))
				currentCmdSuccessful = false
				continue
			}
			// Convert ratio (0-1) to mm. Max lift height is ~45mm. Min is ~23mm.
			// This needs to be calibrated with actual SDK values if they differ.
			// A simple linear mapping for now:
			const minLiftMm float32 = 0.0 // Assuming 0.0 in SDK means fully down.
			const maxLiftMm float32 = 45.0 // Assuming 1.0 in SDK means fully up (approx 45mm from some sources)
			heightMm := minLiftMm + float32(heightRatio)*(maxLiftMm-minLiftMm)

			logger.Println(fmt.Sprintf("SDK Call: robot.Conn.SetLiftHeight(HeightMm: %f)", heightMm))
			_, errH := robot.Conn.SetLiftHeight(ctx, &vectorpb.SetLiftHeightRequest{HeightMm: heightMm, DurationSec: 0.5}) // Added a short duration
			if errH != nil {
				logger.Println("Error executing SetLiftHeight: ", errH)
				currentCmdSuccessful = false
			}
            if currentCmdSuccessful { time.Sleep(500 * time.Millisecond) }


		case "SET_HEAD_ANGLE":
			if len(cmd.Args) != 1 {
				logger.Println("SET_HEAD_ANGLE expects 1 argument, got ", cmd.Args)
				currentCmdSuccessful = false
				continue
			}
			angle, err := strconv.ParseFloat(cmd.Args[0], 32)
			if err != nil {
				logger.Println(fmt.Sprintf("Error parsing SET_HEAD_ANGLE arguments: %v", err))
				currentCmdSuccessful = false
				continue
			}
			logger.Println(fmt.Sprintf("SDK Call: robot.Conn.SetHeadAngle(AngleDeg: %f)", float32(angle)))
			// Max head angle ~45 deg, Min ~-22 deg
			_, errA := robot.Conn.SetHeadAngle(ctx, &vectorpb.SetHeadAngleRequest{AngleDeg: float32(angle), DurationSec: 0.5}) // Added a short duration
			if errA != nil {
				logger.Println("Error executing SetHeadAngle: ", errA)
				currentCmdSuccessful = false
			}
            if currentCmdSuccessful { time.Sleep(500 * time.Millisecond) }


		case "PLAY_ANIMATION":
			if len(cmd.Args) != 1 {
				logger.Println("PLAY_ANIMATION expects 1 argument, got ", cmd.Args)
				currentCmdSuccessful = false
				continue
			}
			animName := cmd.Args[0]
			logger.Println(fmt.Sprintf("SDK Call: robot.Conn.PlayAnimation(Name: \"%s\")", animName))
			_, err := robot.Conn.PlayAnimation(ctx, &vectorpb.PlayAnimationRequest{Animation: &vectorpb.Animation{Name: animName}, Loops: 1})
			if err != nil {
				logger.Println("Error executing PlayAnimation: ", err)
				currentCmdSuccessful = false
			}
			// Animation duration varies. May need a way for LLM to specify wait or use CMD_WAIT.
            // For now, just a small fixed delay after starting one.
            if currentCmdSuccessful { time.Sleep(500 * time.Millisecond) }


		case "STOP_AUTONOMY":
			logger.Println("CMD_STOP_AUTONOMY received, signaling loop to stop.")
			select {
			case stopLoop <- true: // Signal the main loop to terminate
			default:
				logger.Println("Failed to send stop signal to main loop, channel might be unbuffered or already signaled.")
			}
			return fmt.Errorf("autonomy stop commanded by LLM") // Stop processing further commands in this batch

		default:
			logger.Println("Unknown command in ExecuteParsedCommands: ", cmd.Name)
			// Optionally, return an error or log. For now, just logs and continues.
		}
		if !currentCmdSuccessful {
			logger.Println("Command execution failed for: ", cmd.Name, " - stopping current batch of commands.")
			// return fmt.Errorf("failed to execute command %s", cmd.Name) // Option: stop all on first error
		}
		// Small delay between commands if multiple are issued by LLM in one go
		time.Sleep(100 * time.Millisecond)
	}
	return nil
}

func absFloat32(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}


func AutonomousLLMLoop(req interface{}, esn string, isKG bool /* unused for now */) (string, error) {
	logger.Println("AutonomousLLMLoop initiated for ESN: ", esn)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	robot, err := vars.GetRobot(esn)
	if err != nil {
		logger.Println("AutonomousLLMLoop: Failed to get robot instance: ", err)
		return "Failed to get robot instance", err
	}
	_, connErr := robot.Conn.BatteryState(ctx, &vectorpb.BatteryStateRequest{})
	if connErr != nil {
		logger.Println("AutonomousLLMLoop: Robot connection check failed: ", connErr)
		return "Robot connection failed", connErr
	}
	logger.Println("AutonomousLLMLoop: Robot connected successfully.")

	behaviorCtrlStart := make(chan bool)
	behaviorCtrlStop := make(chan bool) // For BControl to release
	loopStopSignal := make(chan bool, 1)   // Buffered, to signal the main loop to stop
	interruptStopSignal := make(chan bool, 1) // For InterruptKGSimWhenTouchedOrWaked to stop its own goroutine

	// Acquire behavior control
	go BControl(robot, ctx, behaviorCtrlStart, behaviorCtrlStop)
	select {
	case <-behaviorCtrlStart:
		logger.Println("AutonomousLLMLoop: Behavior control acquired.")
	case <-time.After(5 * time.Second):
		logger.Println("AutonomousLLMLoop: Timeout acquiring behavior control.")
		return "Timeout acquiring behavior control", fmt.Errorf("behavior control timeout")
	}

	// Setup interrupt handler (touch/wake word)
	go func() {
		interrupted := InterruptKGSimWhenTouchedOrWaked(robot, behaviorCtrlStop, interruptStopSignal) // Pass behaviorCtrlStop to allow interrupt to release control
		if interrupted {
			logger.Println("AutonomousLLMLoop: Interrupt signal received (touch/wake).")
			select {
			case loopStopSignal <- true:
			default:
			}
		}
	}()

	var c *openai.Client
	// Setup OpenAI client based on configuration
	if vars.APIConfig.Knowledge.Provider == "openai" {
		c = openai.NewClient(vars.APIConfig.Knowledge.Key)
	} else if vars.APIConfig.Knowledge.Provider == "together" {
		conf := openai.DefaultConfig(vars.APIConfig.Knowledge.Key)
		conf.BaseURL = "https://api.together.xyz/v1"
		c = openai.NewClientWithConfig(conf)
	} else if vars.APIConfig.Knowledge.Provider == "custom" {
		conf := openai.DefaultConfig(vars.APIConfig.Knowledge.Key)
		conf.BaseURL = vars.APIConfig.Knowledge.Endpoint
		c = openai.NewClientWithConfig(conf)
	} else {
		logger.Println("AutonomousLLMLoop: No valid LLM provider configured.")
		behaviorCtrlStop <- true // Release control
        select {
		case interruptStopSignal <- true: // Tell interrupt handler to stop
		default:
		}
		return "No LLM provider configured", fmt.Errorf("no LLM provider")
	}
	logger.Println("AutonomousLLMLoop: LLM client configured for provider: ", vars.APIConfig.Knowledge.Provider)

	// Main autonomous loop
	for {
		select {
		case <-loopStopSignal:
			logger.Println("AutonomousLLMLoop: Stop signal received. Exiting.")
			// Behavior control should have been released by the interrupt handler or STOP_AUTONOMY command
			// but ensure it's signaled for release if loop breaks here.
			select {
			case behaviorCtrlStop <- true:
			default:
			}
			select {
			case interruptStopSignal <- true:
			default:
			}
			return "Autonomous mode stopped.", nil
		case <-ctx.Done():
			logger.Println("AutonomousLLMLoop: Context cancelled. Exiting.")
			select {
			case behaviorCtrlStop <- true:
			default:
			}
			select {
			case interruptStopSignal <- true:
			default:
			}
			return "Autonomous mode context cancelled.", nil
		default:
			// Proceed with loop iteration
		}

		// 1. Get Camera Image (Placeholder - will be skipped if imgErr != nil)
		imageData, imgErr := getCameraImage(robot, ctx)
		if imgErr != nil {
			logger.Println("AutonomousLLMLoop: Error getting camera image: ", imgErr)
		}

		// 2. Prepare LLM Request
		aiReq := CreateAutonomousAIReq(imageData, esn, "") // Pass empty modelName for default

		logger.Println("AutonomousLLMLoop: Sending request to LLM...")
		resp, err := c.CreateChatCompletion(ctx, aiReq)
		if err != nil {
			logger.Println("AutonomousLLMLoop: LLM API call error: ", err)
			time.Sleep(3 * time.Second) // Wait before retrying
			continue
		}

		if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
			logger.Println("AutonomousLLMLoop: LLM returned empty or no choices.")
			time.Sleep(2 * time.Second)
			continue
		}
		llmResponseText := resp.Choices[0].Message.Content
		logger.Println("AutonomousLLMLoop: LLM Response: -->\n", llmResponseText, "\n<--")

		// 4. Parse LLM Response
		commands, parseErr := ParseLLMCommands(llmResponseText)
		if parseErr != nil {
			logger.Println("AutonomousLLMLoop: Error parsing LLM commands: ", parseErr, " Raw response: ", llmResponseText)
			sayText(robot, "I am a bit confused.") // Make Vector say something
			time.Sleep(2 * time.Second)
			continue
		}
		if len(commands) == 0 {
            logger.Println("AutonomousLLMLoop: LLM response parsed into zero commands. LLM might be thinking or just chatting.")
			// If the LLM response wasn't empty but contained no *commands*, it might have been conversational.
			// We could make Vector say the non-command response here.
			if strings.TrimSpace(llmResponseText) != "" {
				sayText(robot, llmResponseText)
			}
            time.Sleep(1 * time.Second)
            continue
        }


		// 5. Execute Commands
		execErr := ExecuteParsedCommands(robot, commands, ctx, behaviorCtrlStop, loopStopSignal) // Pass loopStopSignal
		if execErr != nil {
			logger.Println("AutonomousLLMLoop: Error executing parsed commands: ", execErr)
			if execErr.Error() == "autonomy stop commanded by LLM" {
				// Loop will terminate at the next iteration due to loopStopSignal
			} else {
				sayText(robot, "I had a problem with that.")
			}
		}

		// Delay before next cycle, configurable or fixed
		// Example: Use Temperature as a proxy for "creativity" affecting cycle time
		delayMs := 1000 + int(vars.APIConfig.Knowledge.Temperature*1000)
		if delayMs < 500 { delayMs = 500 }
		if delayMs > 5000 { delayMs = 5000 }
		time.Sleep(time.Duration(delayMs) * time.Millisecond)
	}
}
