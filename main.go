package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"log"
	"net/http"
	"os"
	"io"

	"github.com/gorilla/websocket"
	"google.golang.org/genai"
	"gemini-live/mytools"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // In production, refine this
	},
}

var MODEL_ID2 = "google/gemini-live-2.5-flash-native-audio"
const MODEL_ID3 = "gemini-2.5-flash-native-audio-preview-12-2025"
const MODEL_ID4 = "gemini-3.1-flash-live-preview"

var PROMPT = "Your are a helpful assistant. Your name is Jane."

type HTTPResponse struct {	
	Reponse string
	Value int
}
type SetupPrompt struct {	
	Prompt string
}

const (
	ModelName = MODEL_ID4; // "gemini-2.5-flash"
)

func openLog(fileName string) {
	logFile, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("Error opening log file: %v", err)
	}

	// Set the output of the logger to our log file.
	log.SetOutput(logFile)	

}
	
func main() {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEMINI_API_KEY environment variable is required")
	}

	openLog("gemini-live.log")
	
	http.HandleFunc("/", serveHome)
	http.HandleFunc("/prompt", servePrompt)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(w, r, apiKey)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Server starting on http://localhost:%s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	log.Println("Serving file: index.html")
	http.ServeFile(w, r, "index.html")
}

// stringHandler handles incoming HTTP requests on the /prompt endpoint.
func servePrompt(w http.ResponseWriter, r *http.Request) {
	
	// We only want to handle POST requests for this endpoint.
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is accepted", http.StatusMethodNotAllowed)
		return
	}

	// Read the entire body of the incoming request.
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		http.Error(w, "Could not read request body", http.StatusInternalServerError)
		return
	}
	// It's important to close the request body.
	defer r.Body.Close()

	// Print the received data to the console.
	// We convert the byte slice 'body' to a string.
	log.Printf("Received prompt from browser: %s\n", string(body))
    prompt := SetupPrompt{}
    json.Unmarshal(body, &prompt)
	fmt.Printf("New Prompt: %s\n", prompt.Prompt)
	PROMPT =  prompt.Prompt
	
	// To prevent Cross-Origin Resource Sharing (CORS) errors when running the
	// HTML file locally, we need to add this header.
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Send a response back to the browser confirming receipt.
	// fmt.Fprintln(w, "Data received successfully by Go server!")

    data := HTTPResponse{Reponse: "Got prompt OK"}
    jsonData, err := json.Marshal(data)
	if err != nil {log.Println("cannot marshall json")}
	w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    w.Write(jsonData)

}


func testLive(session *genai.Session) {

	var turnComplete = true;
	var question = "Hi, what is your name?";
	err :=  session.SendClientContent(genai.LiveClientContentInput{
                                        Turns: []*genai.Content{
                                                                {
                                                                        Role: "user",
                                                                        Parts: []*genai.Part{
                                                                                {Text: question},
                                                                        },
                                                                },
                                                        },
                                                        TurnComplete: &turnComplete,
                                                });

        if err != nil {
                log.Printf("Failed to send text to model: %v", err)
                return
        }
}

// simple function to trace what is in the genai Message
func printMessage(msg *genai.LiveServerMessage) {
	currentTime := time.Now()
	strTime := currentTime.Format("2006-01-02 15:04:05")
	
	// ServerContent *LiveServerContent `json:"serverContent,omitempty"`
	if (msg.ServerContent != nil) {fmt.Println("ServerContent: ", strTime, msg.ServerContent)}
	
	// *LiveServerToolCall `json:"toolCall,omitempty"`
	if (msg.ToolCall != nil) {
		fmt.Println("ToolCall: ", strTime, msg.ToolCall)
		fctCalls := msg.ToolCall.FunctionCalls
		// using for loop
		for i := 0; i < len(fctCalls); i++ {
			fmt.Println(fctCalls[i])
		}
	} 
	
	if (msg.ToolCallCancellation != nil) {fmt.Println("ToolCallCancellation: ", strTime)}

	// Optional. Usage metadata about model response(s).
	// UsageMetadata `json:"usageMetadata,omitempty"`
	if (msg.UsageMetadata != nil) {fmt.Println("UsageMetaData: ", strTime, msg.UsageMetadata)}

	// Optional. Voice activity detection signal. Allowlisted only.
	if (msg.VoiceActivityDetectionSignal != nil) {fmt.Println("VoiceActivityDetectionSignal: ", strTime, msg.VoiceActivityDetectionSignal)}
	
	// Optional. Voice activity signal.
	if (msg.VoiceActivity != nil) {fmt.Println("VoiceActivity: ", strTime, msg.VoiceActivity)}
	
	return
}


// call function requested by model
func callFunction(session *genai.Session, fc *genai.FunctionCall) {
	fmt.Printf("  → tool: %q  args: %v\n", fc.Name, fc.Args)

	var result string
	var responses []*genai.FunctionResponse
	
	log.Printf("Calling function: %s\n", fc.Name)
	
	switch fc.Name {
		case "getWeather":
			// Extract the "city" argument.
			// fc.Args is map[string]any, so a type-assert is needed.
			city, _ := fc.Args["city"].(string)
			if city == "" {
				city = "unknown"
			}
			//result = getWeather(city)
			result = fmt.Sprintf("Today's weather in %s will be sunny, with an average temporature of 22 degree Celcius", city)
		case "getDeliveryByNumber":
			number, _ := fc.Args["number"].(string)
			if number == "" {
				number = "unknown"
			}
			result = mytools.FindPackageByNumber(number)
		default:
			result = fmt.Sprintf("unknown tool: %s", fc.Name)
		}

		fmt.Printf("     result: %s\n", result)

		// Build the FunctionResponse.
		// The ID must echo back the ID from the FunctionCall (required
		// when using the Gemini Developer API; safe to include always).
		responses = append(responses, &genai.FunctionResponse{
			ID:   fc.ID, // echo the call ID back
			Name: fc.Name,
			Response: map[string]any{
				"result": result,
			},
		})

	log.Printf("Function result: %s\n", result)

	// Send all responses back in a single SendToolResponse call.
	// The model will resume speaking once it receives these.
	err := session.SendToolResponse(genai.LiveToolResponseInput{
		FunctionResponses: responses,
	})
	if err != nil {
		log.Fatalf("SendToolResponse: %v", err)
	}

}


func handleWebSocket(w http.ResponseWriter, r *http.Request, apiKey string) {
	fmt.Println("Creating websocket ", r)
	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer wsConn.Close()

	getWeatherFunc := genai.FunctionDeclaration{
		Name:        "getWeather",
		Description: "Get the current weather for a given city.",
		Parameters: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"city": {
					Type:        genai.TypeString,
					Description: "The city name, e.g., 'Paris', 'London'.",
				},
			},
			Required: []string{"city"},
		},
	}

	getDeliveryByNumber := genai.FunctionDeclaration{
                Name:        "getDeliveryByNumber",
                Description: "Get package delivery information, using the package number.",
                Parameters: &genai.Schema{
                        Type: genai.TypeObject,
                        Properties: map[string]*genai.Schema{
                                "number": {
                                        Type:        genai.TypeString,
                                        Description: "The package or delivery number, e.g., '4567', '1233'.",
                                },
                        },
                        Required: []string{"number"},
                },
        }

	fmt.Println("Creating GenAI client...")
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		log.Printf("Failed to create GenAI client: %v", err)
		return
	}
	log.Println("genAI Client created!");
	fmt.Println("genAI Client created!");
	
	// modality := []genai.Modality{"AUDIO", "TEXT"};
	modality := []genai.Modality{"AUDIO"};
	var part = genai.Part{Text: PROMPT};
	parts := []*genai.Part{&part};
	systemInstruct := genai.Content{Parts: parts};
	weatherTool := genai.Tool{
		FunctionDeclarations: []*genai.FunctionDeclaration{&getWeatherFunc},
	}
	deliveryTool := genai.Tool{
		FunctionDeclarations: []*genai.FunctionDeclaration{&getDeliveryByNumber},
	}
	listTools := []*genai.Tool{&weatherTool, &deliveryTool}
	config := &genai.LiveConnectConfig{
		ResponseModalities: modality,
		Tools: listTools,
	}
	config.SystemInstruction = &systemInstruct;

	log.Println("Trying to connect to gemini live!");
	session, err := client.Live.Connect(ctx, ModelName, config)
	if err != nil {
		log.Printf("Failed to connect to Gemini Live: %v", err)
		return
	}
	log.Println("Connected to gemini live!");
	fmt.Println("Connected to gemini live!");
	// testLive(session);
	
	defer session.Close()

	// Error channel for goroutines
	errChan := make(chan error, 2)

	// Goroutine 1: Browser -> Gemini
	go func() {
		for {
			messageType, data, err := wsConn.ReadMessage()
			// log.Printf("Got msg from browser");
			if err != nil {
				errChan <- fmt.Errorf("WebSocket read error: %w", err)
				return
			}

			blob := genai.Blob{MIMEType: "audio/pcm;rate=24000", Data: data};
			if messageType == websocket.BinaryMessage {
				// fmt.Println("Getting binary message on websocket");
				// Send raw audio to Gemini
				err = session.SendRealtimeInput(genai.LiveRealtimeInput{
					Audio: &blob,
					},
				);
				if err != nil {
					errChan <- fmt.Errorf("Gemini send error: %w", err)
					return
				}
			} else if messageType == websocket.TextMessage {
			// Potential text input from user
				log.Println("getting text message on websocket");
				var textMsg map[string]string
				if err := json.Unmarshal(data, &textMsg); err == nil {
					if text, ok := textMsg["text"]; ok {
						var turnComplete = true;
						err = session.SendClientContent(genai.LiveClientContentInput{
							Turns: []*genai.Content{
								{
									Role: "user",
									Parts: []*genai.Part{
										{Text: text},
									},
								},
							},
							TurnComplete: &turnComplete,
						})
						if err != nil {
							errChan <- fmt.Errorf("Gemini send text error: %w", err)
							return
						}
					}
				}
			}
		}
	}()

	// Goroutine 2: Gemini -> Browser
	go func() {
		for {
			resp, err := session.Receive()
			log.Println("Got something from gemini");
			if err != nil {
				errChan <- fmt.Errorf("Gemini receive error: %w", err)
				return
			}

			// *LiveServerToolCall `json:"toolCall,omitempty"`
			if (resp.ToolCall != nil) {
				fmt.Println("ToolCall: ", resp.ToolCall)
				fctCalls := resp.ToolCall.FunctionCalls
				// assuming only one tool
				fmt.Println("Tool needed: ", fctCalls[0])
				callFunction(session, fctCalls[0])
				continue
			}
			
			// printMessage(resp)
			if resp.ServerContent != nil {
				if resp.ServerContent.ModelTurn != nil {
					for _, part := range resp.ServerContent.ModelTurn.Parts {
						// fmt.Println("Part: ", part)
						
						if part.InlineData != nil {
							// Forward audio to browser
							err = wsConn.WriteMessage(websocket.BinaryMessage, part.InlineData.Data)
							if err != nil {
								errChan <- fmt.Errorf("WebSocket write audio error: %w", err)
								return
							}
						}
						if part.Text != "" {
							// Forward text to browser
							log.Println("got part text");
							msg := map[string]string{"text": part.Text}
							jsonMsg, _ := json.Marshal(msg)
							log.Println("got part text: ", msg);
							err = wsConn.WriteMessage(websocket.TextMessage, jsonMsg)
							if err != nil {
								errChan <- fmt.Errorf("WebSocket write text error: %w", err)
								return
							}
						}
					}
				}
				if resp.ServerContent.InputTranscription  != nil {
					fmt.Println("Found input")
					transcript := resp.ServerContent.InputTranscription
					fmt.Println("Input: ", transcript.Text, transcript.Finished )
				}
				
			}
		}
	}()

	// Wait for any error
	err = <-errChan
	log.Printf("Session closed: %v", err)
}
