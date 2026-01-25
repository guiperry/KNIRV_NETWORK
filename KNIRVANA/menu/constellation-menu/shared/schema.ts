import { pgTable, text, serial, integer, boolean } from "drizzle-orm/pg-core";
import { createInsertSchema } from "drizzle-zod";
import { z } from "zod";

export const gameSettings = pgTable("game_settings", {
  id: serial("id").primaryKey(),
  musicVolume: integer("music_volume").notNull().default(80),
  sfxVolume: integer("sfx_volume").notNull().default(80),
  graphicsQuality: text("graphics_quality", { enum: ["low", "medium", "high", "ultra"] }).notNull().default("high"),
  language: text("language").notNull().default("en"),
  isFirstLaunch: boolean("is_first_launch").notNull().default(true),
});

export const insertGameSettingsSchema = createInsertSchema(gameSettings);

export type GameSettings = typeof gameSettings.$inferSelect;
export type InsertGameSettings = z.infer<typeof insertGameSettingsSchema>;
export type UpdateGameSettings = Partial<InsertGameSettings>;
