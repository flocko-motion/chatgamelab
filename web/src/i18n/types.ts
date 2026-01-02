export interface TranslationResources {
  common: {
    welcome: string;
    hello: string;
    goodbye: string;
    loading: string;
    error: string;
    save: string;
    cancel: string;
    delete: string;
    edit: string;
    create: string;
    search: string;
    settings: string;
    profile: string;
    logout: string;
    login: string;
    register: string;
    language: string;
    theme: string;
    dark: string;
    light: string;
    system: string;
  };
  navigation: {
    home: string;
    dashboard: string;
    games: string;
    create: string;
    play: string;
    learn: string;
    debug: string;
  };
  game: {
    createNew: string;
    title: string;
    description: string;
    start: string;
    pause: string;
    resume: string;
    restart: string;
    score: string;
    level: string;
    time: string;
    players: string;
    invite: string;
    join: string;
    leave: string;
  };
  errors: {
    network: string;
    server: string;
    validation: string;
    unauthorized: string;
    forbidden: string;
    notFound: string;
    generic: string;
  };
}

export type TranslationKey = keyof TranslationResources[keyof TranslationResources];
