// Mock data for development - activated with ?mock=true URL parameter
// Replace the placeholder data below with real examples from your API responses

export const mockResponses = {
  // User endpoints
  '/user': {
    "id": 1,
    "name": "Florian Metzger-Noel",
    "openaiKeyPublish": "sk-...88Qy",
    "openaiKeyPersonal": "sk-...88Qy"
  },

  // Games endpoints  
  '/games': [
    {"id":1,"title":"Globberland","TitleImage":null,"description":"","scenario":"You are a gooble in a world of globber. Mobbles try to sibble you tibbers and you try to gibbot their abbles to teggle the ruggo of tabbor.","sessionStartSyscall":"Describe the first planet, where the player starts","statusFields":[{"name":"Globber Sticks","value":"17"},{"name":"Globber Glue","value":"medium"},{"name":"Tabbor Juice","value":"half a skull"}],"imageStyle":"pop art, crazy colors","sharePlayActive":true,"sharePlayHash":"2RUPKH6ESWQHM","shareEditActive":false,"shareEditHash":"","userId":1,"userName":"Florian Metzger-Noel"},
    {"id":3,"title":"Pixelworld","TitleImage":null,"description":"","scenario":"The player is inside an 80s video game. A scenario very similar to the classic movie \"Tron\". He has to survive the kind of dangers you expect in a video game and find a way to exit this world. The player has 24 hours to leave - otherwise he'll be trapped forever.","sessionStartSyscall":"Tell, how the player gets sucked into the video game world and present the first scene. At game start, give the player some random items for his inventory.","statusFields":[{"name":"Time left","value":"24 hours"},{"name":"Inventory","value":""}],"imageStyle":"pixel art, 80s, video game","sharePlayActive":false,"sharePlayHash":"","shareEditActive":false,"shareEditHash":"","userId":1,"userName":"Florian Metzger-Noel"},
    {"id":4,"title":"Another Game","TitleImage":null,"description":"This is a new game.","scenario":"An adventure in a fantasy world. The player must find a way out of a castle.","sessionStartSyscall":"Introduce the player to the game and write the first scene.","statusFields":[{"name":"Gold","value":"100"},{"name":"Silver","value":"50"},{"name":"Items","value":"Potion, Sword, Mask"}],"imageStyle":"illustration, watercolor, fantastic","sharePlayActive":false,"sharePlayHash":"","shareEditActive":false,"shareEditHash":"","userId":1,"userName":"Florian Metzger-Noel"},
    {"id":68,"title":"New 04953996","TitleImage":null,"description":"This is a new game.","scenario":"An adventure in a fantasy world. The player must find a way out of a castle.","sessionStartSyscall":"Introduce the player to the game and write the first scene.","statusFields":[{"name":"Gold","value":"100"}],"imageStyle":"illustration, watercolor, fantastic","sharePlayActive":true,"sharePlayHash":"","shareEditActive":false,"shareEditHash":"","userId":1,"userName":"Florian Metzger-Noel"},
    {"id":69,"title":"New 04967739","TitleImage":null,"description":"This is a new game.","scenario":"An adventure in a fantasy world. The player must find a way out of a castle.","sessionStartSyscall":"Introduce the player to the game and write the first scene.","statusFields":[{"name":"Gold","value":"100"}],"imageStyle":"illustration, watercolor, fantastic","sharePlayActive":false,"sharePlayHash":"","shareEditActive":false,"shareEditHash":"","userId":1,"userName":"Florian Metzger-Noel"},
    {"id":72,"title":"Sysmin Quest","TitleImage":null,"description":"This is a new game.","scenario":"In this cerebral adventure, the player takes on the role of a legendary system administrator tasked with installing the notoriously complex and whimsically problematic service \"foo-bar-from-hell\" on the server known as \"Mordor\". Set in the vast, interconnected networks of a hyper-realistic yet surreal digital world, players must navigate through labyrinths of legacy code, fend off daemons of corruption, and solve cryptic puzzles rooted in computer science lore. Along the way, they'll encounter eccentric characters, including AI with quirky bugs, and face challenges that test their knowledge, wit, and patience to the extreme.","sessionStartSyscall":"The game begins in the dimly lit, cluttered office of our protagonist, a system administrator revered across digital realms for their prowess and courage. The monitor's glow illuminates the room, casting long shadows as an email pops up with a daunting request: Install \"foo-bar-from-hell\" on \"Mordor\", a server known for its unforgiving environment and the complexity of tasks it demands. As the player accepts the challenge, they're whisked away into the server's depths, greeted by the ASCII art of the \"Gates of Mordor\", behind which lies the most convoluted system architecture ever conceived. \"Welcome, brave soul,\" a text prompt flickers. \"Your quest is formidable, your path strewn with trials unknown. Shall you accept your fate, or succumb to the abyss of system failure?\n\nMake sure to explain, that \"Mordor\" is a server and that your quest is to install the service \"foo-bar-from-hell\" on that server.","statusFields":[{"name":"System Health","value":"100%"},{"name":"Sanity","value":"100%"},{"name":"Files Downloaded","value":"None"},{"name":"Passwords Stored","value":"None"}],"imageStyle":"illustrations from WIRED magazin, modern, cyberpunky","sharePlayActive":true,"sharePlayHash":"OQ2LVFDN2MUOQ","shareEditActive":false,"shareEditHash":"","userId":1,"userName":"Florian Metzger-Noel"}
  ],

  // Game CRUD operations
  '/game/new': {
    // TODO: Add real new game response here
    id: 'new-game-123',
    name: 'New Mock Game',
    shareState: 'private',
    ownerName: 'Mock User'
  },

  // Individual game data (will match any /game/{id} pattern)
  '/game/:id': {
    "id":1,
    "title":"Globberland",
    "TitleImage":null,
    "description":"",
    "scenario":"You are a gooble in a world of globber. Mobbles try to sibble you tibbers and you try to gibbot their abbles to teggle the ruggo of tabbor.",
    "sessionStartSyscall":"Describe the first planet, where the player starts",
    "statusFields":[{"name":"Globber Sticks","value":"17"},{"name":"Globber Glue","value":"medium"},{"name":"Tabbor Juice","value":"half a skull"}],
    "imageStyle":"pop art, crazy colors",
    "sharePlayActive":true,
    "sharePlayHash":"2RUPKH6ESWQHM",
    "shareEditActive":false,
    "shareEditHash":"",
    "userId":1,
    "userName":"Florian Metzger-Noel"
  },

  // Session endpoints
  '/session/new': {
    "id":3670,
    "gameId":3,
    "userId":1,
    "assistantId":"asst_SLfiFpJdtysXmycEO5Y6qEDH",
    "assistantInstructions":"You are a text-adventure API. You get inputs, what the player wants to do. You act as the game master and decide, what happens. You decide, what's possible and what's not possible - not the player.\nIf the player posts an action, that doesn't work in the world you are simulating, then continue the story with the player failing in his attempt.\nYou're job is not to please the player, but to create a coherent world. You're job is to create a world, that is fun to explore. You're job is to create a world, that is fun to play in.\n\nThe game frontend sends player actions together with player status as json. Example:\n\n{\"type\":\"player-action\",\"action\":\"drink the potion\",\"status\":[{\"name\":\"gold\",\"value\":\"100\"},{\"name\":\"items\",\"value\":\"sword, potion\"}]}\n\nPossible action types are: \nplayer-action: action, which the player wants to do\nintro: system starts a new game session, message contains instructions generating the first scene\n\nWhen you receive a player action, you continue the story based on his actions and update the player status.\n\nYou always answer with a result json. The result json must exactly follow the format of this Example:\n\n{\"chapterId\":0,\"sessionHash\":\"\",\"type\":\"\",\"story\":\"You drink the potion. You feel a little bit dizzy. You feel a little bit stronger.\",\"status\":[{\"name\":\"gold\",\"value\":\"100\"},{\"name\":\"items\",\"value\":\"sword\"}],\"image\":\"a castle in the background, green grass, late afternoon\"}\n\nAs you see in the example, you have to update the status after each player action. The \"image\" field describes the new scenery for a generative image AI to produce artwork.\n\nThe language and literary style ouf your output should follow the scenario definition.\n\nThe JSON structure, field names, etc. are fixed and must not be changed or translated. The image description should be in english always.\nAny changes to the JSON structure will break the game frontend.\n\nYou always stay in your role. You are the game master. You are the world. You are the narrator. You are the storyteller. You decide, what's possible and what not. You are the text-adventure engine. You are the game. Don't please the player, challenge him.\n\nThe scenario:\n\nThe player is inside an 80s video game. A scenario very similar to the classic movie \"Tron\". He has to survive the kind of dangers you expect in a video game and find a way to exit this world. The player has 24 hours to leave - otherwise he'll be trapped forever.\n",
    "threadId":"thread_pZ6rDmHWbAADS6HH5MOLGxL6",
    "hash":"UxZQusKzioRDbDTMX4e8kQ",
    "model":"gpt-4o-mini"
  },

  '/session/:hash': 'GENERATE_DYNAMIC_SESSION_RESPONSE',

  // Note: /public/* endpoints automatically use the same data as their private counterparts
  // by stripping the /public/ prefix in getMockResponse()

  // Auth testing endpoint
  '/external': {
    // TODO: Add real external response here
    message: 'Mock external API response'
  }
};

// Dynamic session response generator for varied gameplay testing
const generateSessionResponse = () => {
  const stories = [
    "You step forward into the pulsing data stream. The digital current carries you through a tunnel of flowing code.",
    "A security program materializes before you, its geometric form crackling with electric energy. It challenges your right to be here.",
    "You discover a hidden data port in the circuit board floor. Strange symbols flicker across its surface, waiting to be decoded.",
    "The digital landscape shifts around you as the system detects your presence. New pathways open while others close.",
    "A friendly AI sprite approaches, offering cryptic advice about the exit portal's location. But can you trust its pixelated smile?",
    "You encounter a maze of laser barriers blocking your path. Each beam pulses with deadly energy that could delete you instantly.",
    "Memory fragments from the real world flash before your eyes. You remember you have limited time before being trapped forever.",
    "A data storm erupts around you, raining cascading code. You must find shelter or risk corruption of your digital form.",
    "You find an abandoned user terminal still logged in. Its screen displays mysterious coordinates and a countdown timer.",
    "The ground beneath you transforms into a moving conveyor of light. You're being carried toward an unknown destination."
  ];

  const statusUpdates = [
    [{"name": "Time left", "value": "23 hours 45 min"}, {"name": "Inventory", "value": "Digital key, Data fragment"}],
    [{"name": "Time left", "value": "23 hours 30 min"}, {"name": "Inventory", "value": "Digital key, Security token, Virus scanner"}],
    [{"name": "Time left", "value": "23 hours 15 min"}, {"name": "Inventory", "value": "Corrupted file, Memory core"}],
    [{"name": "Time left", "value": "23 hours"}, {"name": "Inventory", "value": "Digital key, Power cell, Shield program"}],
    [{"name": "Time left", "value": "22 hours 45 min"}, {"name": "Inventory", "value": "Access code, Navigation data"}]
  ];

  const images = [
    "neon-lit data tunnel, streaming code, electric blue lighting, 80s computer graphics",
    "geometric security program, angular robot guardian, glowing red eyes, cyberpunk arena",
    "hidden data port, glowing symbols, circuit board patterns, mysterious interface",
    "shifting digital landscape, morphing geometric platforms, electric purple sky",
    "friendly AI sprite, pixelated character, helpful guide, retro game aesthetics",
    "laser barrier maze, deadly energy beams, geometric patterns, danger zone",
    "memory fragments, floating screens, nostalgic images, surreal digital space",
    "data storm, cascading binary code rain, chaotic digital weather, green matrix",
    "abandoned terminal, old computer interface, mysterious coordinates, vintage CRT glow",
    "moving light conveyor, flowing energy pathways, dynamic digital transportation"
  ];

  const randomIndex = Math.floor(Math.random() * stories.length);
  const chapterId = Math.floor(Math.random() * 20) + 1;
  
  return {
    "chapterId": chapterId,
    "sessionHash": "UxZQusKzioRDbDTMX4e8kQ",
    "type": "story",
    "story": stories[randomIndex],
    "status": statusUpdates[randomIndex % statusUpdates.length],
    "image": images[randomIndex],
    "error": "",
    "rawInput": `{"type":"player-action","action":"mock player action","status":[]}`,
    "rawOutput": `{"chapterId":${chapterId},"story":"${stories[randomIndex].substring(0, 50)}...","type":"story"}`,
    "assistantInstructions": "Mock session response generator for design testing",
    "agent": {
      "key": "..fi8V",
      "model": "gpt-4o-mini",
      "assistant": "asst_SLfiFpJdtysXmycEO5Y6qEDH", 
      "thread": "thread_pZ6rDmHWbAADS6HH5MOLGxL6",
      "computationTime": `${(Math.random() * 5 + 2).toFixed(3)}s`
    }
  };
};

// Helper to match dynamic routes (with :id, :hash parameters)
export const getMockResponse = (endpoint, data = null, method = 'GET') => {
  console.log(`[MOCK MODE] ${method} ${endpoint}`, data);

  // Strip /public/ prefix - public endpoints use same data as private ones
  let lookupEndpoint = endpoint;
  if (endpoint.startsWith('/public/')) {
    lookupEndpoint = endpoint.replace('/public/', '/');
    console.log(`[MOCK MODE] Stripped public prefix: ${endpoint} -> ${lookupEndpoint}`);
  }

  // Direct match first
  if (mockResponses[lookupEndpoint]) {
    const response = mockResponses[lookupEndpoint];
    if (response === 'GENERATE_DYNAMIC_SESSION_RESPONSE') {
      return Promise.resolve(generateSessionResponse());
    }
    return Promise.resolve(response);
  }

  // Pattern matching for dynamic routes
  for (const [pattern, response] of Object.entries(mockResponses)) {
    if (pattern.includes(':')) {
      const regex = new RegExp('^' + pattern.replace(/:[^/]+/g, '[^/]+') + '$');
      if (regex.test(lookupEndpoint)) {
        if (response === 'GENERATE_DYNAMIC_SESSION_RESPONSE') {
          return Promise.resolve(generateSessionResponse());
        }
        return Promise.resolve(response);
      }
    }
  }

  // Fallback for unmatched endpoints
  console.warn(`[MOCK MODE] No mock response found for: ${endpoint}`);
  return Promise.resolve({ error: `Mock response not implemented for ${endpoint}` });
};