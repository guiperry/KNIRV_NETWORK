use bevy::prelude::*;
use bevy_rapier3d::prelude::*;
use std::collections::HashMap;

mod game_engine;
mod components;
mod systems;
mod resources;
mod networking;
mod mobile;

use game_engine::*;
use components::*;
use systems::*;
use resources::*;
use networking::*;
use mobile::*;

#[cfg(target_os = "android")]
use android_logger::Config;

fn main() {
    #[cfg(target_os = "android")]
    android_logger::init_once(Config::default().with_min_level(log::Level::Info));

    let mut app = App::new();

    // Configure for mobile or desktop
    #[cfg(feature = "mobile")]
    {
        app.add_plugins(DefaultPlugins.set(WindowPlugin {
            primary_window: Some(Window {
                title: "KNIRVANA".to_string(),
                resolution: (800.0, 600.0).into(),
                resizable: true,
                ..default()
            }),
            ..default()
        }));
    }

    #[cfg(not(feature = "mobile"))]
    {
        app.add_plugins(DefaultPlugins.set(WindowPlugin {
            primary_window: Some(Window {
                title: "KNIRVANA - Desktop".to_string(),
                resolution: (1920.0, 1080.0).into(),
                resizable: true,
                ..default()
            }),
            ..default()
        }));
    }

    app
        // Physics
        .add_plugins(RapierPhysicsPlugin::<NoUserData>::default())
        .add_plugins(RapierDebugRenderPlugin::default())

        // Game plugins
        .add_plugins(GameEnginePlugin)
        .add_plugins(NetworkingPlugin)
        .add_plugins(MobilePlugin)

        // Resources
        .init_resource::<GameConfig>()
        .init_resource::<GameState>()
        .init_resource::<PlayerData>()
        .init_resource::<NetworkManager>()

        // Systems
        .add_systems(Startup, (
            setup_game_world,
            setup_camera,
            setup_lighting,
            setup_ui,
        ))
        .add_systems(Update, (
            player_movement_system,
            camera_follow_system,
            npc_ai_system,
            interaction_system,
            challenge_system,
            network_sync_system,
            mobile_input_system,
            ui_update_system,
        ))

        .run();
}
