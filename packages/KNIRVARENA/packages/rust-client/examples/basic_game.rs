use bevy::prelude::*;
use knirvana_game_client::*;

fn main() {
    println!("Starting KNIRVANA Game Client Example");
    
    App::new()
        .add_plugins(DefaultPlugins.set(WindowPlugin {
            primary_window: Some(Window {
                title: "KNIRVANA - Basic Example".to_string(),
                resolution: (1280.0, 720.0).into(),
                resizable: true,
                ..default()
            }),
            ..default()
        }))
        .add_plugins(GameEnginePlugin)
        .add_systems(Startup, setup_example)
        .add_systems(Update, example_system)
        .run();
}

fn setup_example(mut commands: Commands) {
    info!("Setting up basic KNIRVANA example");
    
    // Spawn a simple camera
    commands.spawn(Camera3dBundle {
        transform: Transform::from_xyz(0.0, 5.0, 10.0)
            .looking_at(Vec3::ZERO, Vec3::Y),
        ..default()
    });
    
    // Add some basic lighting
    commands.insert_resource(AmbientLight {
        color: Color::WHITE,
        brightness: 0.5,
    });
}

fn example_system(time: Res<Time>) {
    // Simple example system that logs every 5 seconds
    if time.elapsed_seconds() as u32 % 5 == 0 && time.delta_seconds() < 0.1 {
        info!("KNIRVANA game client running... Elapsed: {:.1}s", time.elapsed_seconds());
    }
}
