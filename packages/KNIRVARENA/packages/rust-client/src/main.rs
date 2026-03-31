use bevy::prelude::*;
use bevy_rapier3d::prelude::*;

mod game_engine;
mod components;
mod systems;
mod resources;
mod networking;
mod mobile;
mod knirvana_systems;
mod knirvana_ui;
mod nrn_economics;

use game_engine::*;
use components::*;
use systems::{setup_camera, setup_lighting};
use resources::*;
use networking::*;
use mobile::*;
use knirvana_systems::*;
use knirvana_ui::*;
use nrn_economics::*;

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
        .init_resource::<GameMetrics>()
        .init_resource::<MobileSettings>()

        // KNIRVANA-specific resources
        .init_resource::<KnirvanaGameState>()
        .init_resource::<PlayerResources>()
        .init_resource::<KnirvGraphState>()
        .init_resource::<CompetitiveState>()
        .init_resource::<AgentManager>()
        .init_resource::<VisualEffectsState>()
        .init_resource::<UIState>()
        .init_resource::<NRNTokenManager>()
        .init_resource::<NRNPricing>()

        // Systems
        .add_systems(Startup, (
            setup_knirvana_world,
            setup_camera,
            setup_lighting,
            setup_knirvana_ui,
            spawn_initial_agents,
        ))
        // KNIRVANA core systems
        .add_systems(Update, (
            animate_tron_effects,
            update_progress_indicators,
            handle_selection_system,
            spawn_error_nodes_system,
            camera_follow_system,
        ))

        // Graph visualization systems
        .add_systems(Update, (
            update_graph_connections,
            animate_data_flow,
            update_data_flow_particles,
            update_graph_bounds,
        ))

        // Agent and game mechanics
        .add_systems(Update, (
            agent_deployment_system,
            calculate_agent_efficiency_system,
            update_agent_thoughts_system,
            skill_node_generation_system,
            competitive_resolution_system,
        ))

        // Multiplayer competitive systems
        .add_systems(Update, (
            multiplayer_competition_system,
            update_leaderboard_system,
            multiplayer_sync_system,
            progress_comparison_system,
        ))

        // UI systems
        .add_systems(Update, (
            update_nrn_display,
            update_game_stats,
            update_agent_panel,
            update_error_node_info,
            update_thought_display,
            handle_ui_visibility,
            update_notifications,
        ))

        // NRN economics systems
        .add_systems(Update, (
            nrn_consumption_system,
            nrn_bounty_system,
            agent_deployment_cost_system,
            dynamic_pricing_system,
            blockchain_integration_system,
        ))

        // Mobile optimization systems
        .add_systems(Update, (
            mobile_agent_optimization,
            handle_orientation_change,
            mobile_haptic_feedback,
        ))

        .run();
}
