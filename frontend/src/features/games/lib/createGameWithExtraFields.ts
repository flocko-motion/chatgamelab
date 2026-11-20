import type { Game } from "@/api/generated/api.gen";
import type { CreateGameFormData } from "../types";

/**
 * Create a game and optionally update it with extra fields (scenario, game start, image style, status fields, theme).
 * This two-step pattern is needed because the create endpoint only accepts basic fields.
 *
 * Used by AllGames, MyGames, GamesManagement, and useWorkshopGames.
 *
 * @param data - Form data for the new game
 * @param createMutateAsync - `createGame.mutateAsync` from useCreateGame
 * @param updateMutateAsync - `updateGame.mutateAsync` from useUpdateGame
 */
export async function createGameWithExtraFields(
  data: CreateGameFormData,
  createMutateAsync: (req: {
    name: string;
    description: string;
    public: boolean;
    genre?: string;
  }) => Promise<Game>,
  updateMutateAsync: (req: { id: string; game: Game }) => Promise<Game>,
): Promise<Game> {
  const newGame = await createMutateAsync({
    name: data.name,
    description: data.description,
    public: data.isPublic,
    genre: data.genre,
  });

  const hasExtraFields =
    data.systemMessageScenario ||
    data.systemMessageGameStart ||
    data.imageStyle ||
    data.statusFields ||
    data.theme;

  if (newGame.id && hasExtraFields) {
    await updateMutateAsync({
      id: newGame.id,
      game: {
        ...newGame,
        systemMessageScenario: data.systemMessageScenario ?? "",
        systemMessageGameStart: data.systemMessageGameStart ?? "",
        imageStyle: data.imageStyle ?? "",
        statusFields: data.statusFields ?? "",
        theme: data.theme,
      },
    });
  }

  return newGame;
}
