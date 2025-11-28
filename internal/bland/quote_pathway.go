package bland

// ProjectQuotePathway creates a comprehensive pathway for collecting software project quotes.
// This pathway guides the AI through a structured conversation to gather all
// necessary information for generating accurate project quotes.
func ProjectQuotePathway(webhookURL, businessName string) *CreatePathwayRequest {
	return &CreatePathwayRequest{
		Name:        "QuickQuote Software Project Collection",
		Description: "Structured conversation flow for collecting software project requirements",
		Nodes:       projectPathwayNodes(webhookURL, businessName),
		Edges:       projectPathwayEdges(),
	}
}

// projectPathwayNodes returns all nodes for the project quote collection pathway.
func projectPathwayNodes(webhookURL, businessName string) []PathwayNode {
	return []PathwayNode{
		// ============================================
		// ENTRY POINT: Greeting
		// ============================================
		{
			ID:   "greeting",
			Name: "Welcome Greeting",
			Type: "default",
			Data: &NodeData{
				Prompt: `Welcome the caller warmly to ` + businessName + `. Thank them for calling and ask how you can help them today with their software project. Be friendly, professional, and conversational.`,
				Variables: []NodeVariable{
					{Name: "caller_intent", Type: "string", Description: "What the caller wants help with"},
				},
			},
			Position: &NodePosition{X: 0, Y: 0},
		},

		// ============================================
		// PROJECT TYPE SELECTION
		// ============================================
		{
			ID:   "identify_project_type",
			Name: "Identify Project Type",
			Type: "default",
			Data: &NodeData{
				Prompt: `Ask the caller what type of software project they need help with. The common types are:
- Web Application (websites, web apps, portals)
- Mobile App (iOS, Android, or cross-platform)
- API / Backend Services
- E-commerce / Online Store
- Custom Software / Enterprise System
- Integration / Automation

If they're not sure, help them figure out which type best fits their needs. Extract the project type they want.`,
				Variables: []NodeVariable{
					{Name: "project_type", Type: "string", Description: "The type of project: web_app, mobile_app, api, ecommerce, custom_software, integration", Required: true},
				},
				DecisionGuide: `Based on the project_type:
- "web_app" -> go to web_app_details
- "mobile_app" -> go to mobile_app_details
- "api" -> go to api_details
- "ecommerce" -> go to ecommerce_details
- "custom_software" -> go to custom_software_details
- "integration" -> go to integration_details
- unclear -> stay here and clarify`,
			},
			Position: &NodePosition{X: 0, Y: 100},
		},

		// ============================================
		// WEB APPLICATION FLOW
		// ============================================
		{
			ID:   "web_app_details",
			Name: "Web App - Project Details",
			Type: "default",
			Data: &NodeData{
				Prompt: `Collect information about the web application they need. Ask for:
1. What is the main purpose of the web app?
2. Who are the target users (internal team, customers, public)?
3. Do they have any existing website or system to integrate with?
4. Roughly how many pages or sections do they envision?

Be conversational - you can ask multiple questions naturally.`,
				Variables: []NodeVariable{
					{Name: "project_purpose", Type: "string", Description: "Main purpose of the web app", Required: true},
					{Name: "target_users", Type: "string", Description: "Who will use the app: internal, customers, public"},
					{Name: "existing_systems", Type: "string", Description: "Existing websites or systems to integrate"},
					{Name: "estimated_scope", Type: "string", Description: "Rough size estimate: small, medium, large"},
				},
			},
			Position: &NodePosition{X: -300, Y: 200},
		},
		{
			ID:   "web_app_features",
			Name: "Web App - Key Features",
			Type: "default",
			Data: &NodeData{
				Prompt: `Ask about the key features they need:
1. User accounts and authentication?
2. Payment processing or subscriptions?
3. Content management (blog, products, etc.)?
4. Dashboards or reporting?
5. Third-party integrations (email, CRM, etc.)?

Note which features are must-haves vs nice-to-haves.`,
				Variables: []NodeVariable{
					{Name: "needs_auth", Type: "boolean", Description: "Needs user authentication"},
					{Name: "needs_payments", Type: "boolean", Description: "Needs payment processing"},
					{Name: "needs_cms", Type: "boolean", Description: "Needs content management"},
					{Name: "needs_dashboard", Type: "boolean", Description: "Needs dashboards or reporting"},
					{Name: "key_integrations", Type: "string", Description: "Third-party integrations needed"},
				},
			},
			Position: &NodePosition{X: -300, Y: 300},
		},

		// ============================================
		// MOBILE APP FLOW
		// ============================================
		{
			ID:   "mobile_app_details",
			Name: "Mobile App - Project Details",
			Type: "default",
			Data: &NodeData{
				Prompt: `Collect information about the mobile app they need:
1. What does the app need to do? What problem does it solve?
2. Which platforms - iOS, Android, or both?
3. Does it need to work offline?
4. Will it connect to any existing backend or API?`,
				Variables: []NodeVariable{
					{Name: "project_purpose", Type: "string", Description: "Main purpose of the mobile app", Required: true},
					{Name: "platforms", Type: "string", Description: "Target platforms: ios, android, both", Required: true},
					{Name: "needs_offline", Type: "boolean", Description: "Needs offline functionality"},
					{Name: "backend_integration", Type: "string", Description: "Backend/API to integrate with"},
				},
			},
			Position: &NodePosition{X: -100, Y: 200},
		},
		{
			ID:   "mobile_app_features",
			Name: "Mobile App - Features",
			Type: "default",
			Data: &NodeData{
				Prompt: `Ask about specific mobile app features:
1. Push notifications?
2. Camera or photo functionality?
3. Location services or maps?
4. In-app purchases or subscriptions?
5. Social media integration or sharing?

Understand what's critical vs optional.`,
				Variables: []NodeVariable{
					{Name: "needs_push", Type: "boolean", Description: "Needs push notifications"},
					{Name: "needs_camera", Type: "boolean", Description: "Needs camera functionality"},
					{Name: "needs_location", Type: "boolean", Description: "Needs location services"},
					{Name: "needs_iap", Type: "boolean", Description: "Needs in-app purchases"},
					{Name: "needs_social", Type: "boolean", Description: "Needs social integration"},
				},
			},
			Position: &NodePosition{X: -100, Y: 300},
		},

		// ============================================
		// API / BACKEND FLOW
		// ============================================
		{
			ID:   "api_details",
			Name: "API - Project Details",
			Type: "default",
			Data: &NodeData{
				Prompt: `Collect information about the API or backend service:
1. What data or functionality will the API provide?
2. What systems will consume this API (mobile apps, web apps, partners)?
3. Do you have existing data that needs to be migrated?
4. Any specific performance requirements (high traffic, real-time)?`,
				Variables: []NodeVariable{
					{Name: "api_purpose", Type: "string", Description: "What the API provides", Required: true},
					{Name: "api_consumers", Type: "string", Description: "Systems that will use the API"},
					{Name: "data_migration", Type: "boolean", Description: "Needs data migration"},
					{Name: "performance_needs", Type: "string", Description: "Performance requirements"},
				},
			},
			Position: &NodePosition{X: 100, Y: 200},
		},

		// ============================================
		// E-COMMERCE FLOW
		// ============================================
		{
			ID:   "ecommerce_details",
			Name: "E-commerce - Project Details",
			Type: "default",
			Data: &NodeData{
				Prompt: `Collect information about the e-commerce project:
1. What will you be selling (physical products, digital, services)?
2. Approximately how many products or SKUs?
3. Do you need inventory management?
4. Which payment methods (credit cards, PayPal, etc.)?
5. Any specific shipping or tax requirements?`,
				Variables: []NodeVariable{
					{Name: "product_type", Type: "string", Description: "What they're selling: physical, digital, services", Required: true},
					{Name: "product_count", Type: "string", Description: "Approximate number of products"},
					{Name: "needs_inventory", Type: "boolean", Description: "Needs inventory management"},
					{Name: "payment_methods", Type: "string", Description: "Required payment methods"},
					{Name: "shipping_needs", Type: "string", Description: "Shipping or fulfillment requirements"},
				},
			},
			Position: &NodePosition{X: 300, Y: 200},
		},

		// ============================================
		// CUSTOM SOFTWARE FLOW
		// ============================================
		{
			ID:   "custom_software_details",
			Name: "Custom Software - Project Details",
			Type: "default",
			Data: &NodeData{
				Prompt: `Collect information about the custom software project:
1. What business problem are you trying to solve?
2. What processes does this software need to handle?
3. How many users will need access?
4. Are there any compliance requirements (HIPAA, SOC2, etc.)?`,
				Variables: []NodeVariable{
					{Name: "business_problem", Type: "string", Description: "The business problem to solve", Required: true},
					{Name: "key_processes", Type: "string", Description: "Main processes to handle"},
					{Name: "user_count", Type: "string", Description: "Expected number of users"},
					{Name: "compliance_needs", Type: "string", Description: "Compliance requirements"},
				},
			},
			Position: &NodePosition{X: 500, Y: 200},
		},

		// ============================================
		// INTEGRATION FLOW
		// ============================================
		{
			ID:   "integration_details",
			Name: "Integration - Project Details",
			Type: "default",
			Data: &NodeData{
				Prompt: `Collect information about the integration or automation project:
1. What systems need to be connected?
2. What data needs to flow between them?
3. Does this need to run automatically or be triggered?
4. How often does the data need to sync?`,
				Variables: []NodeVariable{
					{Name: "systems_to_connect", Type: "string", Description: "Systems that need integration", Required: true},
					{Name: "data_flow", Type: "string", Description: "What data flows between systems"},
					{Name: "trigger_type", Type: "string", Description: "Automatic or manual trigger"},
					{Name: "sync_frequency", Type: "string", Description: "How often data syncs"},
				},
			},
			Position: &NodePosition{X: 700, Y: 200},
		},

		// ============================================
		// TIMELINE AND BUDGET (COMMON)
		// ============================================
		{
			ID:   "timeline_budget",
			Name: "Timeline and Budget",
			Type: "default",
			Data: &NodeData{
				Prompt: `Now ask about their timeline and budget:
1. When do they need this completed? Is there a hard deadline?
2. Do they have a budget range in mind?
3. Is this a one-time project or will they need ongoing support?

Be understanding - many people aren't sure about budget yet.`,
				Variables: []NodeVariable{
					{Name: "target_deadline", Type: "string", Description: "When they need it completed"},
					{Name: "budget_range", Type: "string", Description: "Their budget range if provided"},
					{Name: "ongoing_support", Type: "boolean", Description: "Need ongoing support after launch"},
				},
			},
			Position: &NodePosition{X: 0, Y: 400},
		},

		// ============================================
		// CONTACT INFORMATION (COMMON)
		// ============================================
		{
			ID:   "collect_contact_info",
			Name: "Contact Information",
			Type: "default",
			Data: &NodeData{
				Prompt: `Now collect their contact information so we can send them their quote:
1. Their full name
2. Best phone number to reach them
3. Email address (for sending the quote details)
4. Company name (if applicable)

Confirm each piece of information by repeating it back.`,
				Variables: []NodeVariable{
					{Name: "full_name", Type: "string", Description: "Caller's full name", Required: true},
					{Name: "phone_number", Type: "string", Description: "Best contact phone number", Required: true},
					{Name: "email", Type: "string", Description: "Email address for quote delivery", Required: true},
					{Name: "company_name", Type: "string", Description: "Company name if applicable"},
				},
			},
			Position: &NodePosition{X: 0, Y: 500},
		},

		// ============================================
		// SUMMARY AND CONFIRMATION
		// ============================================
		{
			ID:   "summarize_quote_request",
			Name: "Summarize Request",
			Type: "default",
			Data: &NodeData{
				Prompt: `Provide a brief summary of all the information collected:
- The type of project they need
- Key features and requirements
- Their timeline and budget expectations
- Their contact information

Ask if everything is correct and if they'd like to change anything.`,
				Variables: []NodeVariable{
					{Name: "summary_confirmed", Type: "boolean", Description: "Customer confirmed the summary is correct", Required: true},
				},
			},
			Position: &NodePosition{X: 0, Y: 600},
		},

		// ============================================
		// SUBMIT QUOTE REQUEST (WEBHOOK)
		// ============================================
		{
			ID:   "submit_quote",
			Name: "Submit Quote Request",
			Type: "webhook",
			Data: &NodeData{
				WebhookURL:      webhookURL + "/quote-request",
				WebhookMethod:   "POST",
				PreWebhookText:  "Let me submit your project details now. This will just take a moment.",
				PostWebhookText: "Your project quote request has been submitted successfully!",
			},
			Position: &NodePosition{X: 0, Y: 700},
		},

		// ============================================
		// CLOSING
		// ============================================
		{
			ID:   "closing",
			Name: "Closing",
			Type: "end_call",
			Data: &NodeData{
				EndMessage: `Thank them for calling ` + businessName + `. Let them know:
1. They'll receive their detailed quote within 24-48 hours by email
2. A project consultant may follow up with any clarifying questions
3. They can call back anytime if they have questions

Wish them a great day and end the call warmly.`,
			},
			Position: &NodePosition{X: 0, Y: 800},
		},

		// ============================================
		// GLOBAL NODES (Accessible from any node)
		// ============================================
		{
			ID:   "faq_handler",
			Name: "FAQ Handler",
			Type: "default",
			Data: &NodeData{
				Prompt: `The caller has a question. Answer it helpfully using available knowledge, then guide them back to where they were in the quote process.`,
				IsGlobal: true,
			},
			Position: &NodePosition{X: 300, Y: 0},
		},
		{
			ID:   "transfer_to_agent",
			Name: "Transfer to Agent",
			Type: "transfer",
			Data: &NodeData{
				TransferMessage: "I'll connect you with one of our project consultants who can help you further. Please hold for just a moment.",
				IsGlobal:        true,
			},
			Position: &NodePosition{X: 300, Y: 50},
		},
		{
			ID:   "not_interested",
			Name: "Not Interested",
			Type: "end_call",
			Data: &NodeData{
				EndMessage: "Thank them for their time and let them know they can call back anytime. End politely.",
				IsGlobal:   true,
			},
			Position: &NodePosition{X: 300, Y: 100},
		},
	}
}

// projectPathwayEdges returns all edges connecting nodes in the project quote collection pathway.
func projectPathwayEdges() []PathwayEdge {
	return []PathwayEdge{
		// Entry flow
		NewEdge("greeting", "identify_project_type", "Continue", "User wants a project quote"),

		// Web app flow
		NewEdge("identify_project_type", "web_app_details", "Web Application", "project_type is web_app"),
		NewEdge("web_app_details", "web_app_features", "Continue", "Project details collected"),
		NewEdge("web_app_features", "timeline_budget", "Continue", "Features collected"),

		// Mobile app flow
		NewEdge("identify_project_type", "mobile_app_details", "Mobile App", "project_type is mobile_app"),
		NewEdge("mobile_app_details", "mobile_app_features", "Continue", "Project details collected"),
		NewEdge("mobile_app_features", "timeline_budget", "Continue", "Features collected"),

		// API flow
		NewEdge("identify_project_type", "api_details", "API / Backend", "project_type is api"),
		NewEdge("api_details", "timeline_budget", "Continue", "API details collected"),

		// E-commerce flow
		NewEdge("identify_project_type", "ecommerce_details", "E-commerce", "project_type is ecommerce"),
		NewEdge("ecommerce_details", "timeline_budget", "Continue", "E-commerce details collected"),

		// Custom software flow
		NewEdge("identify_project_type", "custom_software_details", "Custom Software", "project_type is custom_software"),
		NewEdge("custom_software_details", "timeline_budget", "Continue", "Custom software details collected"),

		// Integration flow
		NewEdge("identify_project_type", "integration_details", "Integration", "project_type is integration"),
		NewEdge("integration_details", "timeline_budget", "Continue", "Integration details collected"),

		// Common closing flow
		NewEdge("timeline_budget", "collect_contact_info", "Continue", "Timeline and budget discussed"),
		NewEdge("collect_contact_info", "summarize_quote_request", "Continue", "Contact info collected"),
		NewEdge("summarize_quote_request", "submit_quote", "Confirmed", "summary_confirmed is true"),
		NewEdge("submit_quote", "closing", "Complete", "Quote submitted successfully"),

		// Correction flow
		NewEdge("summarize_quote_request", "identify_project_type", "Start Over", "User wants to change project type"),
		NewEdge("summarize_quote_request", "collect_contact_info", "Fix Contact", "User wants to correct contact info"),

		// Global edges (can be reached from multiple nodes)
		NewEdge("greeting", "not_interested", "Not Interested", "User is not interested"),
		NewEdge("identify_project_type", "transfer_to_agent", "Speak to Agent", "User wants to speak to a person"),
	}
}

// DefaultProjectKnowledgeBase returns text content for a knowledge base
// containing common software project information.
func DefaultProjectKnowledgeBase() string {
	return `# Software Project Information Knowledge Base

## Project Types

### Web Applications
- **Single Page Applications (SPAs)**: Modern, interactive apps built with React, Vue, or Angular
- **Progressive Web Apps (PWAs)**: Web apps that work offline and can be installed
- **Server-Side Rendered**: Traditional web apps with better SEO
- **Portals & Dashboards**: Business intelligence and data visualization

### Mobile Applications
- **Native iOS**: Built specifically for iPhone/iPad using Swift
- **Native Android**: Built for Android devices using Kotlin/Java
- **Cross-Platform**: React Native or Flutter apps that work on both platforms
- **Hybrid**: Web technologies wrapped in a native container

### E-commerce Solutions
- **Shopify/WooCommerce**: Quick-to-market with existing platforms
- **Custom E-commerce**: Full control over the shopping experience
- **Marketplace**: Multi-vendor platforms like Etsy or Amazon
- **Subscription/SaaS**: Recurring billing and member management

### APIs & Backend Services
- **REST APIs**: Standard web APIs for mobile and web apps
- **GraphQL**: Flexible query-based APIs
- **Real-time Services**: WebSockets, chat, live updates
- **Microservices**: Scalable, distributed architecture

## Common Features & Integrations

### Authentication & Users
- Social login (Google, Facebook, Apple)
- Two-factor authentication
- Role-based access control
- Single sign-on (SSO)

### Payments
- Stripe, PayPal, Square integration
- Subscription management
- Invoicing systems
- Multi-currency support

### Communication
- Email (SendGrid, Mailchimp)
- SMS (Twilio)
- Push notifications
- In-app messaging

### Data & Analytics
- Custom dashboards
- Report generation
- Data export (CSV, PDF)
- Business intelligence integration

## Timeline Expectations

### Small Projects (4-8 weeks)
- Simple websites or landing pages
- Basic mobile apps
- API integrations
- Minor customizations

### Medium Projects (2-4 months)
- Full web applications
- Mobile apps with backend
- E-commerce stores
- Custom business tools

### Large Projects (4-12 months)
- Enterprise software
- Complex platforms
- Multi-system integrations
- Complete digital transformations

## Budget Considerations

### Factors That Affect Cost
- Complexity of features
- Number of integrations
- Design requirements
- Performance needs
- Security/compliance requirements
- Ongoing maintenance needs

### Investment Levels
- Starter: MVP or proof of concept
- Standard: Full-featured solution
- Enterprise: Scalable, secure, fully supported

---
Our team will provide a detailed quote based on your specific requirements.
`
}
