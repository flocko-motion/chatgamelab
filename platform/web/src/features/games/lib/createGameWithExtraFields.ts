import type { ObjGame } from "@/api/generated";
import type { CreateGameFormData } from "../types";

/**
 * Create a game and optionally update it with extra fields (scenario, game start, image style, status fields).
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
  }) => Promise<ObjGame>,
  updateMutateAsync: (req: { id: string; game: ObjGame }) => Promise<ObjGame>,
): Promise<ObjGame> {
  const newGame = await createMutateAsync({
    name: data.name,
    description: data.description,
    public: data.isPublic,
  });

  const hasExtraFields =
    data.systemMessageScenario ||
    data.systemMessageGameStart ||
    data.imageStyle ||
    data.statusFields;

  if (newGame.id && hasExtraFields) {
    await updateMutateAsync({
      id: newGame.id,
      game: {
        ...newGame,
        systemMessageScenario: data.systemMessageScenario,
        systemMessageGameStart: data.systemMessageGameStart,
        imageStyle: data.imageStyle,
        statusFields: data.statusFields,
      },
    });
  }

  return newGame;
}
