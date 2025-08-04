use bevy::prelude::*;
use bevy_rapier3d::prelude::*;
use crate::components::*;
use crate::game_engine::*;
use crate::resources::*;

pub fn setup_game_world(
    mut commands: Commands,
    mut meshes: ResMut<Assets<Mesh>>,
    mut materials: ResMut<Assets<StandardMaterial>>,
    asset_server: Res<AssetServer>,
) {
    info!("Setting up game world");

    // Create terrain
    let terrain_mesh = meshes.add(Plane3d::default().mesh().size(100.0, 100.0));
    let terrain_material = materials.add(StandardMaterial {
        base_color: Color::rgb(0.3, 0.7, 0.3),
        ..default()
    });

    commands.spawn((
        PbrBundle {
            mesh: terrain_mesh,
            material: terrain_material,
            transform: Transform::from_xyz(0.0, 0.0, 0.0),
            ..default()
        },
        Collider::cuboid(50.0, 0.1, 50.0),
        Environment {
            terrain_type: "grassland".to_string(),
            weather: WeatherType::Clear,
            time_of_day: 12.0,
        },
        Name::new("Terrain"),
    ));

    // Spawn player
    spawn_player(&mut commands, &mut meshes, &mut materials);

    // Spawn NPCs
    spawn_npcs(&mut commands, &mut meshes, &mut materials);

    // Spawn interactables
    spawn_interactables(&mut commands, &mut meshes, &mut materials);

    info!("Game world setup complete");
}

pub fn setup_camera(mut commands: Commands) {
    // Main camera
    commands.spawn((
        Camera3dBundle {
            transform: Transform::from_xyz(0.0, 10.0, 15.0)
                .looking_at(Vec3::ZERO, Vec3::Y),
            ..default()
        },
        MainCamera,
        Name::new("Main Camera"),
    ));
}

pub fn setup_lighting(mut commands: Commands) {
    // Ambient light
    commands.insert_resource(AmbientLight {
        color: Color::WHITE,
        brightness: 0.3,
    });

    // Directional light (sun)
    commands.spawn(DirectionalLightBundle {
        directional_light: DirectionalLight {
            color: Color::WHITE,
            illuminance: 10000.0,
            shadows_enabled: true,
            ..default()
        },
        transform: Transform::from_rotation(Quat::from_euler(
            EulerRot::XYZ,
            -std::f32::consts::FRAC_PI_4,
            std::f32::consts::FRAC_PI_4,
            0.0,
        )),
        ..default()
    });
}

pub fn setup_ui(mut commands: Commands) {
    // UI Root
    commands.spawn((
        NodeBundle {
            style: Style {
                width: Val::Percent(100.0),
                height: Val::Percent(100.0),
                ..default()
            },
            ..default()
        },
        Name::new("UI Root"),
    ));
}

fn spawn_player(
    commands: &mut Commands,
    meshes: &mut ResMut<Assets<Mesh>>,
    materials: &mut ResMut<Assets<StandardMaterial>>,
) {
    let player_mesh = meshes.add(Capsule3d::new(0.5, 1.8));
    let player_material = materials.add(StandardMaterial {
        base_color: Color::rgb(0.0, 0.8, 0.0),
        ..default()
    });

    commands.spawn((
        PbrBundle {
            mesh: player_mesh,
            material: player_material,
            transform: Transform::from_xyz(0.0, 1.0, 0.0),
            ..default()
        },
        RigidBody::Dynamic,
        Collider::capsule_y(0.9, 0.5),
        LockedAxes::ROTATION_LOCKED,
        Player {
            id: "player_1".to_string(),
            name: "Player".to_string(),
            skills: vec!["basic_movement".to_string()],
            nrn_balance: 100.0,
            level: 1,
            experience: 0,
        },
        PlayerController {
            move_speed: 5.0,
            jump_force: 10.0,
            is_grounded: false,
        },
        PlayerAvatar,
        Health {
            current: 100.0,
            maximum: 100.0,
            regeneration_rate: 1.0,
        },
        Energy {
            current: 100.0,
            maximum: 100.0,
            consumption_rate: 0.5,
        },
        Inventory {
            items: vec![],
            capacity: 20,
        },
        NetworkSync {
            last_sync: 0.0,
            sync_interval: 0.1,
            dirty: false,
        },
        Name::new("Player"),
    ));
}

fn spawn_npcs(
    commands: &mut Commands,
    meshes: &mut ResMut<Assets<Mesh>>,
    materials: &mut ResMut<Assets<StandardMaterial>>,
) {
    let npc_mesh = meshes.add(Capsule3d::new(0.4, 1.6));
    let npc_material = materials.add(StandardMaterial {
        base_color: Color::rgb(0.8, 0.4, 0.2),
        ..default()
    });

    // Spawn a quest giver NPC
    commands.spawn((
        PbrBundle {
            mesh: npc_mesh.clone(),
            material: npc_material.clone(),
            transform: Transform::from_xyz(5.0, 1.0, 5.0),
            ..default()
        },
        RigidBody::Fixed,
        Collider::capsule_y(0.8, 0.4),
        NPC {
            id: "npc_quest_giver_1".to_string(),
            name: "Elder Sage".to_string(),
            npc_type: "quest_giver".to_string(),
            dialogue: DialogueTree {
                nodes: std::collections::HashMap::new(),
                current_node: "greeting".to_string(),
            },
            skills: vec!["wisdom".to_string(), "teaching".to_string()],
            quest_giver: true,
        },
        NPCAi {
            behavior_type: "stationary".to_string(),
            interaction_range: 3.0,
            last_interaction: 0.0,
        },
        QuestGiver {
            available_quests: vec!["first_steps".to_string()],
            completed_quests: vec![],
        },
        Name::new("Elder Sage"),
    ));
}

fn spawn_interactables(
    commands: &mut Commands,
    meshes: &mut ResMut<Assets<Mesh>>,
    materials: &mut ResMut<Assets<StandardMaterial>>,
) {
    let crystal_mesh = meshes.add(Sphere::new(0.5));
    let crystal_material = materials.add(StandardMaterial {
        base_color: Color::rgb(0.2, 0.8, 1.0),
        emissive: Color::rgb(0.1, 0.4, 0.5),
        ..default()
    });

    // Spawn NRN crystal
    commands.spawn((
        PbrBundle {
            mesh: crystal_mesh,
            material: crystal_material,
            transform: Transform::from_xyz(-5.0, 1.0, -5.0),
            ..default()
        },
        RigidBody::Fixed,
        Collider::ball(0.5),
        Interactable {
            id: "nrn_crystal_1".to_string(),
            interaction_type: "collect".to_string(),
            action: "nrn_reward".to_string(),
            requirements: vec![],
            rewards: vec![Reward {
                reward_type: "nrn".to_string(),
                amount: 10.0,
                item: None,
            }],
        },
        Collectible {
            item_type: "nrn_crystal".to_string(),
            value: 10.0,
            rarity: "common".to_string(),
        },
        Name::new("NRN Crystal"),
    ));
}

pub fn player_movement_system(
    keyboard_input: Res<ButtonInput<KeyCode>>,
    mut player_query: Query<(&mut Transform, &mut Velocity, &PlayerController), With<PlayerAvatar>>,
    time: Res<Time>,
) {
    for (mut transform, mut velocity, controller) in player_query.iter_mut() {
        let mut movement = Vec3::ZERO;

        // Handle keyboard input
        if keyboard_input.pressed(KeyCode::KeyW) || keyboard_input.pressed(KeyCode::ArrowUp) {
            movement.z -= 1.0;
        }
        if keyboard_input.pressed(KeyCode::KeyS) || keyboard_input.pressed(KeyCode::ArrowDown) {
            movement.z += 1.0;
        }
        if keyboard_input.pressed(KeyCode::KeyA) || keyboard_input.pressed(KeyCode::ArrowLeft) {
            movement.x -= 1.0;
        }
        if keyboard_input.pressed(KeyCode::KeyD) || keyboard_input.pressed(KeyCode::ArrowRight) {
            movement.x += 1.0;
        }

        // Normalize and apply movement
        if movement.length() > 0.0 {
            movement = movement.normalize() * controller.move_speed;
            velocity.linvel.x = movement.x;
            velocity.linvel.z = movement.z;
        } else {
            velocity.linvel.x = 0.0;
            velocity.linvel.z = 0.0;
        }

        // Handle jumping
        if keyboard_input.just_pressed(KeyCode::Space) && controller.is_grounded {
            velocity.linvel.y = controller.jump_force;
        }
    }
}

pub fn camera_follow_system(
    player_query: Query<&Transform, (With<PlayerAvatar>, Without<MainCamera>)>,
    mut camera_query: Query<&mut Transform, (With<MainCamera>, Without<PlayerAvatar>)>,
    time: Res<Time>,
) {
    if let Ok(player_transform) = player_query.get_single() {
        if let Ok(mut camera_transform) = camera_query.get_single_mut() {
            let target_position = player_transform.translation + Vec3::new(0.0, 10.0, 15.0);

            // Smooth camera following
            camera_transform.translation = camera_transform.translation.lerp(
                target_position,
                time.delta_seconds() * 2.0,
            );

            // Look at player
            camera_transform.look_at(player_transform.translation, Vec3::Y);
        }
    }
}

pub fn npc_ai_system(
    mut npc_query: Query<(&mut Transform, &NPCAi, &NPC)>,
    player_query: Query<&Transform, (With<PlayerAvatar>, Without<NPC>)>,
    time: Res<Time>,
) {
    if let Ok(player_transform) = player_query.get_single() {
        for (mut npc_transform, ai, npc) in npc_query.iter_mut() {
            let distance = npc_transform.translation.distance(player_transform.translation);

            // Simple AI: Look at player when in range
            if distance < ai.interaction_range * 2.0 {
                let direction = (player_transform.translation - npc_transform.translation).normalize();
                let target_rotation = Quat::from_rotation_y(direction.x.atan2(direction.z));

                npc_transform.rotation = npc_transform.rotation.lerp(
                    target_rotation,
                    time.delta_seconds() * 2.0,
                );
            }
        }
    }
}

pub fn interaction_system(
    keyboard_input: Res<ButtonInput<KeyCode>>,
    player_query: Query<(Entity, &Transform), With<PlayerAvatar>>,
    interactable_query: Query<(Entity, &Transform, &Interactable)>,
    npc_query: Query<(Entity, &Transform, &NPC)>,
    mut interaction_events: EventWriter<InteractionEvent>,
) {
    if keyboard_input.just_pressed(KeyCode::KeyE) {
        if let Ok((player_entity, player_transform)) = player_query.get_single() {
            let interaction_range = 3.0;

            // Check for nearby interactables
            for (entity, transform, interactable) in interactable_query.iter() {
                let distance = player_transform.translation.distance(transform.translation);
                if distance < interaction_range {
                    interaction_events.send(InteractionEvent {
                        player_entity,
                        target_entity: entity,
                        interaction_type: interactable.interaction_type.clone(),
                    });
                    break;
                }
            }

            // Check for nearby NPCs
            for (entity, transform, npc) in npc_query.iter() {
                let distance = player_transform.translation.distance(transform.translation);
                if distance < interaction_range {
                    interaction_events.send(InteractionEvent {
                        player_entity,
                        target_entity: entity,
                        interaction_type: "dialogue".to_string(),
                    });
                    break;
                }
            }
        }
    }
}

pub fn challenge_system(
    mut challenge_events: EventReader<ChallengeEvent>,
    mut challenge_query: Query<&mut Challenge>,
    time: Res<Time>,
) {
    for event in challenge_events.read() {
        info!("Processing challenge event: {} - {}", event.challenge_id, event.event_type);

        for mut challenge in challenge_query.iter_mut() {
            if challenge.id == event.challenge_id {
                match event.event_type.as_str() {
                    "start" => {
                        challenge.is_active = true;
                        info!("Started challenge: {}", challenge.name);
                    }
                    "complete" => {
                        challenge.is_active = false;
                        info!("Completed challenge: {}", challenge.name);
                    }
                    "fail" => {
                        challenge.is_active = false;
                        info!("Failed challenge: {}", challenge.name);
                    }
                    _ => {}
                }
            }
        }
    }
}

pub fn network_sync_system(
    mut network_query: Query<(&mut NetworkSync, &Player)>,
    time: Res<Time>,
    network_manager: Res<NetworkManager>,
) {
    let current_time = time.elapsed_seconds_f64();

    for (mut sync, player) in network_query.iter_mut() {
        if current_time - sync.last_sync > sync.sync_interval {
            if sync.dirty {
                // Sync player data to server
                info!("Syncing player data for: {}", player.name);
                sync.dirty = false;
            }
            sync.last_sync = current_time;
        }
    }
}

pub fn mobile_input_system(
    touches: Res<Touches>,
    mut player_query: Query<(&mut Transform, &mut Velocity, &PlayerController), With<PlayerAvatar>>,
    time: Res<Time>,
) {
    #[cfg(feature = "mobile")]
    {
        for (mut transform, mut velocity, controller) in player_query.iter_mut() {
            // Handle touch input for mobile
            if let Some(touch) = touches.first_pressed_position() {
                // Simple touch-to-move system
                let screen_center = Vec2::new(400.0, 300.0); // Adjust based on screen size
                let touch_delta = touch - screen_center;

                if touch_delta.length() > 50.0 {
                    let movement = Vec3::new(
                        touch_delta.x / 100.0,
                        0.0,
                        -touch_delta.y / 100.0,
                    ).normalize() * controller.move_speed;

                    velocity.linvel.x = movement.x;
                    velocity.linvel.z = movement.z;
                }
            } else {
                velocity.linvel.x = 0.0;
                velocity.linvel.z = 0.0;
            }
        }
    }
}

pub fn ui_update_system(
    player_query: Query<&Player>,
    mut text_query: Query<&mut Text>,
) {
    if let Ok(player) = player_query.get_single() {
        for mut text in text_query.iter_mut() {
            // Update UI with player stats
            text.sections[0].value = format!(
                "Level: {} | NRN: {:.1} | XP: {}",
                player.level, player.nrn_balance, player.experience
            );
        }
    }
}
