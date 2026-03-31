use bevy::prelude::*;
use bevy::render::mesh::shape::UVSphere;
use bevy_rapier3d::prelude::*;
use std::f32::consts::PI;
use crate::components::*;
use crate::resources::*;

// ============================================================================
// KNIRVANA GAME SETUP SYSTEMS
// ============================================================================

/// Setup the KNIRVANA game world with TRON-style environment
pub fn setup_knirvana_world(
    mut commands: Commands,
    mut meshes: ResMut<Assets<Mesh>>,
    mut materials: ResMut<Assets<StandardMaterial>>,
    mut knirv_graph: ResMut<KnirvGraphState>,
) {
    info!("Setting up KNIRVANA world");

    // Create dark TRON-style environment
    setup_tron_environment(&mut commands, &mut meshes, &mut materials);
    
    // Spawn initial ErrorNodes
    spawn_initial_error_nodes(&mut commands, &mut meshes, &mut materials, &mut knirv_graph);
    
    // Spawn initial SkillNodes
    spawn_initial_skill_nodes(&mut commands, &mut meshes, &mut materials, &mut knirv_graph);
    
    // Setup graph connections
    setup_graph_connections(&mut commands, &mut meshes, &mut materials);

    info!("KNIRVANA world setup complete");
}

/// Create the dark TRON-style environment
fn setup_tron_environment(
    commands: &mut Commands,
    meshes: &mut ResMut<Assets<Mesh>>,
    materials: &mut ResMut<Assets<StandardMaterial>>,
) {
    // Dark grid floor
    let grid_mesh = meshes.add(Mesh::from(bevy::render::mesh::shape::Plane { size: 200.0, subdivisions: 20 }));
    let grid_material = materials.add(StandardMaterial {
        base_color: Color::rgb(0.0, 0.05, 0.1),
        emissive: Color::rgb(0.0, 0.1, 0.2),
        metallic: 0.8,
        perceptual_roughness: 0.2,
        ..default()
    });

    commands.spawn((
        PbrBundle {
            mesh: grid_mesh,
            material: grid_material,
            transform: Transform::from_xyz(0.0, -0.1, 0.0),
            ..default()
        },
        Collider::cuboid(100.0, 0.05, 100.0),
        Name::new("TRON Grid Floor"),
    ));
}

/// Spawn initial ErrorNodes in the game world
fn spawn_initial_error_nodes(
    commands: &mut Commands,
    meshes: &mut ResMut<Assets<Mesh>>,
    materials: &mut ResMut<Assets<StandardMaterial>>,
    knirv_graph: &mut ResMut<KnirvGraphState>,
) {
    let error_node_mesh = meshes.add(Mesh::from(UVSphere { radius: 0.5, sectors: 16, stacks: 8 }));
    
    // Create different materials for different error types
    let logic_error_material = materials.add(StandardMaterial {
        base_color: Color::rgb(1.0, 0.2, 0.2),
        emissive: Color::rgb(0.5, 0.1, 0.1),
        metallic: 0.3,
        perceptual_roughness: 0.7,
        ..default()
    });

    let data_error_material = materials.add(StandardMaterial {
        base_color: Color::rgb(1.0, 0.6, 0.0),
        emissive: Color::rgb(0.5, 0.3, 0.0),
        metallic: 0.3,
        perceptual_roughness: 0.7,
        ..default()
    });

    let network_error_material = materials.add(StandardMaterial {
        base_color: Color::rgb(0.8, 0.2, 1.0),
        emissive: Color::rgb(0.4, 0.1, 0.5),
        metallic: 0.3,
        perceptual_roughness: 0.7,
        ..default()
    });

    // Spawn 5 initial ErrorNodes
    for i in 0..5 {
        let angle = (i as f32) * 2.0 * PI / 5.0;
        let radius = 10.0;
        let position = Vec3::new(
            angle.cos() * radius,
            1.0,
            angle.sin() * radius,
        );

        let error_node = ErrorNode {
            id: format!("error_node_{}", i),
            node_type: match i % 3 {
                0 => ErrorNodeType::LogicError,
                1 => ErrorNodeType::DataInconsistency,
                _ => ErrorNodeType::NetworkFailure,
            },
            difficulty: 0.3 + (i as f32) * 0.15,
            bounty: 50.0 + (i as f64) * 25.0,
            progress: 0.0,
            is_being_solved: false,
            solver_agent_id: None,
            description: format!("Error node {} requiring resolution", i + 1),
            created_at: 0.0,
            estimated_time: 30.0 + (i as f32) * 10.0,
        };

        let material = match error_node.node_type {
            ErrorNodeType::LogicError => logic_error_material.clone(),
            ErrorNodeType::DataInconsistency => data_error_material.clone(),
            ErrorNodeType::NetworkFailure => network_error_material.clone(),
            _ => logic_error_material.clone(),
        };

        let entity = commands.spawn((
            PbrBundle {
                mesh: error_node_mesh.clone(),
                material,
                transform: Transform::from_translation(position),
                ..default()
            },
            RigidBody::Fixed,
            Collider::ball(0.5),
            error_node.clone(),
            TronEffect {
                glow_intensity: 1.0,
                pulse_speed: 2.0,
                color: Color::rgb(1.0, 0.2, 0.2),
                is_pulsing: true,
                animation_time: 0.0,
            },
            Selectable {
                is_selected: false,
                is_hovered: false,
                selection_radius: 50.0, // Increased for easier selection
            },
            Interactable {
                interaction_type: InteractionType::DeployAgent,
                interaction_range: 2.0,
                is_available: true,
                cooldown_remaining: 0.0,
            },
            ProgressIndicator {
                current_progress: 0.0,
                target_progress: 0.0,
                animation_speed: 1.0,
                show_percentage: true,
            },
            Name::new(format!("ErrorNode_{}", i)),
        )).id();

        knirv_graph.error_nodes.insert(error_node.id.clone(), entity);
    }
}

/// Spawn initial SkillNodes in the game world
fn spawn_initial_skill_nodes(
    commands: &mut Commands,
    meshes: &mut ResMut<Assets<Mesh>>,
    materials: &mut ResMut<Assets<StandardMaterial>>,
    knirv_graph: &mut ResMut<KnirvGraphState>,
) {
    let skill_node_mesh = meshes.add(Mesh::from(bevy::render::mesh::shape::Box::new(0.6, 0.6, 0.6)));
    
    let skill_material = materials.add(StandardMaterial {
        base_color: Color::rgb(0.0, 1.0, 0.5),
        emissive: Color::rgb(0.0, 0.5, 0.25),
        metallic: 0.8,
        perceptual_roughness: 0.2,
        ..default()
    });

    // Spawn 3 initial SkillNodes
    for i in 0..3 {
        let angle = (i as f32) * 2.0 * PI / 3.0;
        let radius = 15.0;
        let position = Vec3::new(
            angle.cos() * radius,
            1.5,
            angle.sin() * radius,
        );

        let skill_node = SkillNode {
            id: format!("skill_node_{}", i),
            name: format!("Skill {}", i + 1),
            category: match i % 3 {
                0 => SkillCategory::Debugging,
                1 => SkillCategory::Optimization,
                _ => SkillCategory::Security,
            },
            created_by: "system".to_string(),
            usage_count: 0,
            value: 100.0 + (i as f64) * 50.0,
            created_at: 0.0,
            effectiveness: 0.8,
        };

        let entity = commands.spawn((
            PbrBundle {
                mesh: skill_node_mesh.clone(),
                material: skill_material.clone(),
                transform: Transform::from_translation(position),
                ..default()
            },
            RigidBody::Fixed,
            Collider::cuboid(0.3, 0.3, 0.3),
            skill_node.clone(),
            TronEffect {
                glow_intensity: 0.8,
                pulse_speed: 1.5,
                color: Color::rgb(0.0, 1.0, 0.5),
                is_pulsing: false,
                animation_time: 0.0,
            },
            Selectable {
                is_selected: false,
                is_hovered: false,
                selection_radius: 50.0, // Increased for easier selection
            },
            Name::new(format!("SkillNode_{}", i)),
        )).id();

        knirv_graph.skill_nodes.insert(skill_node.id.clone(), entity);
    }
}

/// Setup visual connections between nodes
fn setup_graph_connections(
    commands: &mut Commands,
    meshes: &mut ResMut<Assets<Mesh>>,
    materials: &mut ResMut<Assets<StandardMaterial>>,
) {
    // Connection line material
    let _connection_material = materials.add(StandardMaterial {
        base_color: Color::rgb(0.0, 0.8, 1.0),
        emissive: Color::rgb(0.0, 0.4, 0.5),
        unlit: true,
        ..default()
    });

    // Connections will be created dynamically as nodes are connected
    info!("Graph connection system ready");
}

// ============================================================================
// KNIRVANA RENDERING SYSTEMS
// ============================================================================

/// System to animate TRON effects on game objects
pub fn animate_tron_effects(
    time: Res<Time>,
    mut query: Query<(&mut TronEffect, &mut Transform, &Handle<StandardMaterial>)>,
    mut materials: ResMut<Assets<StandardMaterial>>,
) {
    for (mut tron_effect, mut transform, material_handle) in query.iter_mut() {
        tron_effect.animation_time += time.delta_seconds();

        if tron_effect.is_pulsing {
            let pulse = (tron_effect.animation_time * tron_effect.pulse_speed).sin() * 0.5 + 0.5;
            let scale = 1.0 + pulse * 0.1;
            transform.scale = Vec3::splat(scale);

            // Update material emissive intensity
            if let Some(material) = materials.get_mut(material_handle) {
                let intensity = tron_effect.glow_intensity * (0.5 + pulse * 0.5);
                material.emissive = tron_effect.color * intensity;
            }
        }

        // Rotate ErrorNodes based on their difficulty
        transform.rotation *= Quat::from_rotation_y(0.01);
    }
}

/// System to update progress indicators
pub fn update_progress_indicators(
    time: Res<Time>,
    mut query: Query<(&mut ProgressIndicator, &ErrorNode)>,
) {
    for (mut progress_indicator, error_node) in query.iter_mut() {
        progress_indicator.target_progress = error_node.progress;

        // Smooth animation towards target
        let diff = progress_indicator.target_progress - progress_indicator.current_progress;
        if diff.abs() > 0.001 {
            progress_indicator.current_progress += diff * progress_indicator.animation_speed * time.delta_seconds();
        }
    }
}

/// System to handle mouse input for object selection
pub fn handle_selection_system(
    mut game_state: ResMut<KnirvanaGameState>,
    mut selectable_query: Query<(Entity, &mut Selectable, &Transform, &mut TronEffect, &Handle<StandardMaterial>), Or<(With<ErrorNode>, With<AIAgent>, With<SkillNode>)>>,
    mut materials: ResMut<Assets<StandardMaterial>>,
    mouse_input: Res<Input<MouseButton>>,
    windows: Query<&Window>,
    camera_query: Query<(&Camera, &GlobalTransform), With<Camera3d>>,
) {
    // Debug: Check if system is running and mouse state
    static mut FRAME_COUNT: u32 = 0;
    unsafe {
        FRAME_COUNT += 1;
        if FRAME_COUNT % 60 == 0 { // Log every 60 frames (roughly once per second)
            info!("Selection system running - frame {}, mouse pressed: {}, mouse just_pressed: {}",
                FRAME_COUNT,
                mouse_input.pressed(MouseButton::Left),
                mouse_input.just_pressed(MouseButton::Left)
            );
        }
    }

    // Debug: Log when mouse is clicked
    if mouse_input.just_pressed(MouseButton::Left) {
        info!("Mouse left click detected!");

        if let Ok(window) = windows.get_single() {
            if let Some(cursor_position) = window.cursor_position() {
                info!("Cursor position: {:?}", cursor_position);

                if let Ok((camera, camera_transform)) = camera_query.get_single() {
                    if let Some(ray) = camera.viewport_to_world(camera_transform, cursor_position) {
                        info!("Ray cast from camera: origin={:?}, direction={:?}", ray.origin, ray.direction);

                        let mut closest_entity = None;
                        let mut closest_distance = f32::MAX;
                        let mut entity_count = 0;

                        // Check all selectable entities
                        for (entity, selectable, transform, _, _) in selectable_query.iter() {
                            entity_count += 1;
                            let distance = ray.origin.distance(transform.translation);
                            info!("Entity {:?} at position {:?}, distance: {:.2}, selection_radius: {:.2}",
                                entity, transform.translation, distance, selectable.selection_radius);

                            if distance < selectable.selection_radius && distance < closest_distance {
                                closest_distance = distance;
                                closest_entity = Some(entity);
                                info!("New closest entity: {:?} at distance {:.2}", entity, distance);
                            }
                        }

                        info!("Found {} selectable entities", entity_count);

                        // Update selection state
                        for (entity, mut selectable, _, _, _) in selectable_query.iter_mut() {
                            selectable.is_selected = Some(entity) == closest_entity;
                        }

                        // Update game state based on what was selected
                        if let Some(selected_entity) = closest_entity {
                            // Simple selection logic - could be improved with component checks
                            game_state.selected_error_node = Some(selected_entity);
                            game_state.selected_agent = None;
                            info!("Selected entity: {:?}", selected_entity);
                        } else {
                            // Clear selection
                            game_state.selected_error_node = None;
                            game_state.selected_agent = None;
                            info!("No entity selected - cleared selection");
                        }
                    } else {
                        info!("Failed to create ray from camera");
                    }
                } else {
                    info!("Failed to get camera");
                }
            } else {
                info!("No cursor position available");
            }
        } else {
            info!("Failed to get window");
        }
    }

    // Update visual effects based on selection state
    for (entity, selectable, _, mut tron_effect, material_handle) in selectable_query.iter_mut() {
        if selectable.is_selected {
            tron_effect.glow_intensity = 1.5;
            tron_effect.is_pulsing = true;

            if let Some(material) = materials.get_mut(material_handle) {
                material.emissive = tron_effect.color * 0.8;
            }
        } else if selectable.is_hovered {
            tron_effect.glow_intensity = 1.2;

            if let Some(material) = materials.get_mut(material_handle) {
                material.emissive = tron_effect.color * 0.6;
            }
        } else {
            tron_effect.glow_intensity = 1.0;
            tron_effect.is_pulsing = false;

            if let Some(material) = materials.get_mut(material_handle) {
                material.emissive = tron_effect.color * 0.3;
            }
        }
    }
}

/// System to spawn new ErrorNodes periodically
pub fn spawn_error_nodes_system(
    mut commands: Commands,
    mut meshes: ResMut<Assets<Mesh>>,
    mut materials: ResMut<Assets<StandardMaterial>>,
    mut knirv_graph: ResMut<KnirvGraphState>,
    time: Res<Time>,
) {
    knirv_graph.node_spawn_timer += time.delta_seconds();

    // Spawn new ErrorNode every 30 seconds if under max limit
    if knirv_graph.node_spawn_timer >= 30.0 && knirv_graph.error_nodes.len() < knirv_graph.max_error_nodes as usize {
        knirv_graph.node_spawn_timer = 0.0;

        let error_node_mesh = meshes.add(Mesh::from(UVSphere { radius: 0.5, sectors: 16, stacks: 8 }));
        let error_material = materials.add(StandardMaterial {
            base_color: Color::rgb(1.0, 0.3, 0.3),
            emissive: Color::rgb(0.5, 0.15, 0.15),
            metallic: 0.3,
            perceptual_roughness: 0.7,
            ..default()
        });

        // Random position around the graph center
        let angle = fastrand::f32() * 2.0 * PI;
        let radius = 8.0 + fastrand::f32() * 12.0;
        let position = Vec3::new(
            angle.cos() * radius,
            1.0 + fastrand::f32() * 2.0,
            angle.sin() * radius,
        );

        let node_id = format!("error_node_{}", fastrand::u32(..));
        let error_node = ErrorNode {
            id: node_id.clone(),
            node_type: match fastrand::u32(..) % 6 {
                0 => ErrorNodeType::LogicError,
                1 => ErrorNodeType::DataInconsistency,
                2 => ErrorNodeType::NetworkFailure,
                3 => ErrorNodeType::SecurityVulnerability,
                4 => ErrorNodeType::PerformanceBottleneck,
                _ => ErrorNodeType::IntegrationIssue,
            },
            difficulty: 0.2 + fastrand::f32() * 0.6,
            bounty: 25.0 + (fastrand::f32() * 100.0) as f64,
            progress: 0.0,
            is_being_solved: false,
            solver_agent_id: None,
            description: "Dynamically spawned error requiring resolution".to_string(),
            created_at: time.elapsed_seconds_f64(),
            estimated_time: 20.0 + fastrand::f32() * 60.0,
        };

        let entity = commands.spawn((
            PbrBundle {
                mesh: error_node_mesh,
                material: error_material,
                transform: Transform::from_translation(position),
                ..default()
            },
            RigidBody::Fixed,
            Collider::ball(0.5),
            error_node.clone(),
            TronEffect {
                glow_intensity: 1.0,
                pulse_speed: 2.0,
                color: Color::rgb(1.0, 0.3, 0.3),
                is_pulsing: true,
                animation_time: 0.0,
            },
            Selectable {
                is_selected: false,
                is_hovered: false,
                selection_radius: 1.0,
            },
            Interactable {
                interaction_type: InteractionType::DeployAgent,
                interaction_range: 2.0,
                is_available: true,
                cooldown_remaining: 0.0,
            },
            ProgressIndicator {
                current_progress: 0.0,
                target_progress: 0.0,
                animation_speed: 1.0,
                show_percentage: true,
            },
            Name::new(format!("ErrorNode_{}", node_id)),
        )).id();

        knirv_graph.error_nodes.insert(node_id, entity);
        info!("Spawned new ErrorNode");
    }
}

/// System to update camera to follow selected objects
pub fn camera_follow_system(
    mut camera_query: Query<&mut Transform, (With<Camera3d>, Without<ErrorNode>, Without<AIAgent>)>,
    game_state: Res<KnirvanaGameState>,
    error_node_query: Query<&Transform, (With<ErrorNode>, Without<Camera3d>)>,
    agent_query: Query<&Transform, (With<AIAgent>, Without<Camera3d>)>,
    time: Res<Time>,
) {
    if let Ok(mut camera_transform) = camera_query.get_single_mut() {
        let mut target_position = Vec3::new(0.0, 20.0, 15.0);

        // Follow selected ErrorNode
        if let Some(selected_entity) = game_state.selected_error_node {
            if let Ok(node_transform) = error_node_query.get(selected_entity) {
                target_position = node_transform.translation + Vec3::new(5.0, 10.0, 5.0);
            }
        }
        // Follow selected Agent
        else if let Some(selected_entity) = game_state.selected_agent {
            if let Ok(agent_transform) = agent_query.get(selected_entity) {
                target_position = agent_transform.translation + Vec3::new(3.0, 8.0, 3.0);
            }
        }

        // Smooth camera movement
        let lerp_factor = 2.0 * time.delta_seconds();
        camera_transform.translation = camera_transform.translation.lerp(target_position, lerp_factor);

        // Always look at the center of the graph
        let look_target = game_state.camera_target
            .and_then(|entity| error_node_query.get(entity).ok())
            .map(|transform| transform.translation)
            .unwrap_or(Vec3::ZERO);

        camera_transform.look_at(look_target, Vec3::Y);
    }
}

// ============================================================================
// AGENT DEPLOYMENT AND MANAGEMENT SYSTEMS
// ============================================================================

/// System to spawn initial AI agents for the player
pub fn spawn_initial_agents(
    mut commands: Commands,
    mut meshes: ResMut<Assets<Mesh>>,
    mut materials: ResMut<Assets<StandardMaterial>>,
    mut agent_manager: ResMut<AgentManager>,
    mut knirv_graph: ResMut<KnirvGraphState>,
) {
    let agent_mesh = meshes.add(Mesh::from(UVSphere { radius: 0.3, sectors: 16, stacks: 8 }));

    // Create different materials for different agent types
    let resolver_material = materials.add(StandardMaterial {
        base_color: Color::rgb(0.2, 0.8, 1.0),
        emissive: Color::rgb(0.1, 0.4, 0.5),
        metallic: 0.8,
        perceptual_roughness: 0.2,
        ..default()
    });

    let observer_material = materials.add(StandardMaterial {
        base_color: Color::rgb(1.0, 1.0, 0.2),
        emissive: Color::rgb(0.5, 0.5, 0.1),
        metallic: 0.8,
        perceptual_roughness: 0.2,
        ..default()
    });

    // Spawn 3 initial agents
    for i in 0..3 {
        let agent_type = match i {
            0 => AgentType::Resolver,
            1 => AgentType::Observer,
            _ => AgentType::Helper,
        };

        let specialization = match i {
            0 => SkillCategory::Debugging,
            1 => SkillCategory::Analysis,
            _ => SkillCategory::Optimization,
        };

        let material = match agent_type {
            AgentType::Resolver => resolver_material.clone(),
            _ => observer_material.clone(),
        };

        let position = Vec3::new(
            (i as f32 - 1.0) * 3.0,
            0.5,
            -5.0,
        );

        let agent_id = format!("agent_{}", i);
        let agent = AIAgent {
            id: agent_id.clone(),
            owner_id: "player_1".to_string(),
            agent_type: agent_type.clone(),
            specialization: specialization.clone(),
            efficiency: 0.7 + (i as f32) * 0.1,
            energy: 100.0,
            max_energy: 100.0,
            experience: 0,
            skills: vec![],
            current_task: None,
            status: AgentStatus::Idle,
            thought_process: vec!["Initialized and ready for deployment".to_string()],
            last_action_time: 0.0,
        };

        let entity = commands.spawn((
            PbrBundle {
                mesh: agent_mesh.clone(),
                material,
                transform: Transform::from_translation(position),
                ..default()
            },
            RigidBody::Dynamic,
            Collider::ball(0.3),
            agent.clone(),
            TronEffect {
                glow_intensity: 0.8,
                pulse_speed: 1.0,
                color: Color::rgb(0.2, 0.8, 1.0),
                is_pulsing: false,
                animation_time: 0.0,
            },
            Selectable {
                is_selected: false,
                is_hovered: false,
                selection_radius: 50.0, // Increased for easier selection
            },
            Name::new(format!("Agent_{}", i)),
        )).id();

        knirv_graph.agents.insert(agent_id, entity);
    }

    info!("Spawned initial AI agents");
}

/// System to handle agent deployment to ErrorNodes and keyboard controls
pub fn agent_deployment_system(
    mut commands: Commands,
    mut agent_query: Query<(Entity, &mut AIAgent, &mut Transform), With<AIAgent>>,
    mut error_node_query: Query<(Entity, &mut ErrorNode, &Transform), (With<ErrorNode>, Without<AIAgent>)>,
    mut player_resources: ResMut<PlayerResources>,
    _agent_manager: ResMut<AgentManager>,
    mut game_state: ResMut<KnirvanaGameState>,
    mut ui_state: ResMut<UIState>,
    time: Res<Time>,
    input: Res<Input<KeyCode>>,
) {
    // Handle keyboard controls for UI panels
    if input.just_pressed(KeyCode::Tab) {
        let show_leaderboard = ui_state.show_leaderboard;
        ui_state.show_leaderboard = !show_leaderboard;
        ui_state.notification_queue.push(Notification {
            message: format!("Leaderboard {}", if !show_leaderboard { "shown" } else { "hidden" }),
            notification_type: NotificationType::Info,
            timestamp: time.elapsed_seconds_f64(),
            duration: 2.0,
        });
    }

    if input.just_pressed(KeyCode::A) {
        let show_agent_panel = ui_state.show_agent_panel;
        ui_state.show_agent_panel = !show_agent_panel;
        ui_state.notification_queue.push(Notification {
            message: format!("Agent panel {}", if !show_agent_panel { "shown" } else { "hidden" }),
            notification_type: NotificationType::Info,
            timestamp: time.elapsed_seconds_f64(),
            duration: 2.0,
        });
    }

    if input.just_pressed(KeyCode::I) {
        let show_node_info = ui_state.show_node_info;
        ui_state.show_node_info = !show_node_info;
        ui_state.notification_queue.push(Notification {
            message: format!("Info panel {}", if !show_node_info { "shown" } else { "hidden" }),
            notification_type: NotificationType::Info,
            timestamp: time.elapsed_seconds_f64(),
            duration: 2.0,
        });
    }

    // Handle deployment input (Space key to deploy agent to selected ErrorNode)
    if input.just_pressed(KeyCode::Space) {
        info!("SPACE key pressed!");
        if let Some(selected_error_node) = game_state.selected_error_node {
            info!("Selected error node found: {:?}", selected_error_node);
            // Find an available agent
            let mut available_agent = None;
            let mut agent_count = 0;
            let mut idle_count = 0;
            for (entity, agent, _) in agent_query.iter() {
                agent_count += 1;
                if agent.status == AgentStatus::Idle {
                    idle_count += 1;
                    available_agent = Some(entity);
                    break;
                }
            }
            info!("Found {} total agents, {} idle agents", agent_count, idle_count);

            if let Some(agent_entity) = available_agent {
                // Check if player has enough NRN
                let deployment_cost = 10.0;
                if player_resources.nrn_balance >= deployment_cost {
                    // Deploy the agent
                    if let Ok((_, mut agent, mut agent_transform)) = agent_query.get_mut(agent_entity) {
                        if let Ok((_, mut error_node, error_transform)) = error_node_query.get_mut(selected_error_node) {
                            // Deduct NRN cost
                            player_resources.nrn_balance -= deployment_cost;
                            player_resources.agents_deployed += 1;

                            // Update agent
                            agent.current_task = Some(error_node.id.clone());
                            agent.status = AgentStatus::Working;
                            agent.efficiency = 0.8 + fastrand::f32() * 0.2;
                            agent.thought_process.push(format!("Deploying to {:?} ErrorNode", error_node.node_type));

                            // Move agent to error node position
                            agent_transform.translation = error_transform.translation + Vec3::new(0.0, 2.0, 0.0);

                            // Update error node
                            if !error_node.is_being_solved {
                                error_node.is_being_solved = true;
                                error_node.solver_agent_id = Some(agent.id.clone());
                            }

                            // Add notification
                            ui_state.notification_queue.push(Notification {
                                message: format!("Agent deployed to {:?} error node! Cost: {} NRN",
                                    error_node.node_type, deployment_cost),
                                notification_type: NotificationType::Success,
                                timestamp: time.elapsed_seconds_f64(),
                                duration: 3.0,
                            });

                            info!("Agent {} deployed to ErrorNode {}", agent.id, error_node.id);
                        }
                    }
                } else {
                    // Not enough NRN
                    ui_state.notification_queue.push(Notification {
                        message: format!("Insufficient NRN! Need {} but have {:.1}",
                            deployment_cost, player_resources.nrn_balance),
                        notification_type: NotificationType::Warning,
                        timestamp: time.elapsed_seconds_f64(),
                        duration: 3.0,
                    });
                }
            } else {
                // No available agents
                ui_state.notification_queue.push(Notification {
                    message: "No idle agents available for deployment!".to_string(),
                    notification_type: NotificationType::Warning,
                    timestamp: time.elapsed_seconds_f64(),
                    duration: 3.0,
                });
            }
        } else {
            // No error node selected
            info!("No error node selected when SPACE was pressed");
            ui_state.notification_queue.push(Notification {
                message: "Select an error node first, then press SPACE to deploy an agent!".to_string(),
                notification_type: NotificationType::Info,
                timestamp: time.elapsed_seconds_f64(),
                duration: 3.0,
            });
        }
    }

    // Update agent movement and work progress
    for (agent_entity, mut agent, mut transform) in agent_query.iter_mut() {
        match agent.status {
            AgentStatus::Moving => {
                // Move agent towards target ErrorNode
                if let Some(task_id) = &agent.current_task {
                    for (error_entity, error_node, _) in error_node_query.iter() {
                        if error_node.id == *task_id {
                            // Get error node position (we need to query its transform)
                            let target_pos = Vec3::new(0.0, 0.5, 0.0); // Placeholder
                            let direction = (target_pos - transform.translation).normalize();
                            let move_speed = 5.0;

                            transform.translation += direction * move_speed * time.delta_seconds();

                            // Check if reached target
                            if transform.translation.distance(target_pos) < 1.0 {
                                agent.status = AgentStatus::Working;
                                agent.thought_process.push("Arrived at ErrorNode, beginning analysis".to_string());
                            }
                            break;
                        }
                    }
                }
            },
            AgentStatus::Working => {
                // Agent is working on solving the ErrorNode
                if let Some(task_id) = &agent.current_task {
                    for (error_entity, mut error_node, _) in error_node_query.iter_mut() {
                        if error_node.id == *task_id && error_node.solver_agent_id.as_ref() == Some(&agent.id) {
                            // Calculate progress based on agent efficiency
                            let progress_rate = agent.efficiency * 0.1 * time.delta_seconds();
                            error_node.progress = (error_node.progress + progress_rate).min(1.0);

                            // Update agent thoughts
                            if fastrand::f32() < 0.1 { // 10% chance per frame to add thought
                                let thoughts = vec![
                                    "Analyzing error patterns...",
                                    "Applying debugging techniques...",
                                    "Cross-referencing with known solutions...",
                                    "Optimizing resolution approach...",
                                    "Making progress on the solution...",
                                ];
                                agent.thought_process.push(thoughts[fastrand::usize(..thoughts.len())].to_string());

                                // Keep only last 5 thoughts
                                if agent.thought_process.len() > 5 {
                                    agent.thought_process.remove(0);
                                }
                            }

                            // Check if ErrorNode is solved
                            if error_node.progress >= 1.0 {
                                // Award bounty to player
                                player_resources.nrn_balance += error_node.bounty;
                                player_resources.errors_resolved += 1;
                                player_resources.total_bounty_earned += error_node.bounty;

                                // Update agent
                                agent.experience += 10;
                                agent.current_task = None;
                                agent.status = AgentStatus::Idle;
                                agent.thought_process.push("ErrorNode resolved successfully!".to_string());

                                // Create SkillNode from resolved ErrorNode
                                spawn_skill_node_from_error(&mut commands, &error_node, &agent);

                                info!("ErrorNode {} resolved by agent {}", error_node.id, agent.id);
                            }
                            break;
                        }
                    }
                }
            },
            _ => {}
        }

        // Energy consumption and regeneration
        match agent.status {
            AgentStatus::Working => {
                agent.energy = (agent.energy - 5.0 * time.delta_seconds()).max(0.0);
                if agent.energy <= 0.0 {
                    agent.status = AgentStatus::Resting;
                    agent.thought_process.push("Energy depleted, resting...".to_string());
                }
            },
            AgentStatus::Resting => {
                agent.energy = (agent.energy + 10.0 * time.delta_seconds()).min(agent.max_energy);
                if agent.energy >= agent.max_energy * 0.8 {
                    agent.status = AgentStatus::Idle;
                    agent.thought_process.push("Energy restored, ready for deployment".to_string());
                }
            },
            _ => {
                agent.energy = (agent.energy + 2.0 * time.delta_seconds()).min(agent.max_energy);
            }
        }
    }
}

/// Helper function to spawn a SkillNode when an ErrorNode is resolved
fn spawn_skill_node_from_error(
    commands: &mut Commands,
    error_node: &ErrorNode,
    agent: &AIAgent,
) {
    // This would spawn a new SkillNode based on the resolved ErrorNode
    // Implementation would create the actual SkillNode entity
    info!("Creating SkillNode from resolved ErrorNode: {}", error_node.id);
}

// ============================================================================
// ERRORNODE SOLVING MECHANICS
// ============================================================================

/// System to calculate agent efficiency based on specialization and ErrorNode type
pub fn calculate_agent_efficiency_system(
    mut agent_query: Query<&mut AIAgent>,
    error_node_query: Query<&ErrorNode>,
) {
    for mut agent in agent_query.iter_mut() {
        if let Some(task_id) = &agent.current_task {
            for error_node in error_node_query.iter() {
                if error_node.id == *task_id {
                    // Calculate efficiency bonus based on specialization match
                    let specialization_bonus = match (&agent.specialization, &error_node.node_type) {
                        (SkillCategory::Debugging, ErrorNodeType::LogicError) => 1.5,
                        (SkillCategory::Security, ErrorNodeType::SecurityVulnerability) => 1.5,
                        (SkillCategory::Optimization, ErrorNodeType::PerformanceBottleneck) => 1.5,
                        (SkillCategory::Integration, ErrorNodeType::IntegrationIssue) => 1.5,
                        (SkillCategory::Analysis, ErrorNodeType::DataInconsistency) => 1.5,
                        (SkillCategory::Automation, ErrorNodeType::NetworkFailure) => 1.3,
                        _ => 1.0, // No bonus for mismatched specializations
                    };

                    // Apply difficulty penalty
                    let difficulty_factor = 1.0 - (error_node.difficulty * 0.3);

                    // Calculate final efficiency
                    let base_efficiency = agent.efficiency;
                    let final_efficiency = base_efficiency * specialization_bonus * difficulty_factor;

                    // Update agent efficiency (temporarily for this task)
                    agent.efficiency = final_efficiency.min(2.0); // Cap at 2.0
                    break;
                }
            }
        }
    }
}

/// System to manage real-time thought process display
pub fn update_agent_thoughts_system(
    mut agent_query: Query<&mut AIAgent>,
    error_node_query: Query<&ErrorNode>,
    time: Res<Time>,
) {
    for mut agent in agent_query.iter_mut() {
        if agent.status == AgentStatus::Working {
            if let Some(task_id) = &agent.current_task {
                for error_node in error_node_query.iter() {
                    if error_node.id == *task_id {
                        // Generate contextual thoughts based on progress and error type
                        let progress_stage = if error_node.progress < 0.25 {
                            "analysis"
                        } else if error_node.progress < 0.5 {
                            "planning"
                        } else if error_node.progress < 0.75 {
                            "implementation"
                        } else {
                            "verification"
                        };

                        let thoughts = match (&error_node.node_type, progress_stage) {
                            (ErrorNodeType::LogicError, "analysis") => vec![
                                "Examining code flow patterns...",
                                "Identifying logical inconsistencies...",
                                "Tracing variable state changes...",
                            ],
                            (ErrorNodeType::LogicError, "planning") => vec![
                                "Designing fix strategy...",
                                "Considering edge cases...",
                                "Planning test scenarios...",
                            ],
                            (ErrorNodeType::LogicError, "implementation") => vec![
                                "Implementing logical corrections...",
                                "Refactoring problematic sections...",
                                "Applying best practices...",
                            ],
                            (ErrorNodeType::LogicError, "verification") => vec![
                                "Running validation tests...",
                                "Verifying fix completeness...",
                                "Confirming no regressions...",
                            ],
                            (ErrorNodeType::SecurityVulnerability, "analysis") => vec![
                                "Scanning for attack vectors...",
                                "Analyzing security boundaries...",
                                "Checking input validation...",
                            ],
                            (ErrorNodeType::SecurityVulnerability, "planning") => vec![
                                "Designing security patches...",
                                "Planning access controls...",
                                "Considering threat models...",
                            ],
                            (ErrorNodeType::PerformanceBottleneck, "analysis") => vec![
                                "Profiling execution paths...",
                                "Identifying resource hotspots...",
                                "Measuring latency patterns...",
                            ],
                            _ => vec![
                                "Processing error data...",
                                "Applying resolution techniques...",
                                "Making steady progress...",
                            ],
                        };

                        // Add new thought occasionally
                        if fastrand::f32() < 0.05 { // 5% chance per frame
                            let new_thought = thoughts[fastrand::usize(..thoughts.len())];
                            agent.thought_process.push(new_thought.to_string());

                            // Keep only last 5 thoughts
                            if agent.thought_process.len() > 5 {
                                agent.thought_process.remove(0);
                            }
                        }
                        break;
                    }
                }
            }
        }
    }
}

/// System to automatically generate SkillNodes when ErrorNodes are resolved
pub fn skill_node_generation_system(
    mut commands: Commands,
    mut meshes: ResMut<Assets<Mesh>>,
    mut materials: ResMut<Assets<StandardMaterial>>,
    mut knirv_graph: ResMut<KnirvGraphState>,
    mut player_resources: ResMut<PlayerResources>,
    error_node_query: Query<(Entity, &ErrorNode), Changed<ErrorNode>>,
    agent_query: Query<&AIAgent>,
) {
    for (error_entity, error_node) in error_node_query.iter() {
        // Check if ErrorNode was just completed
        if error_node.progress >= 1.0 && error_node.is_being_solved {
            if let Some(solver_agent_id) = &error_node.solver_agent_id {
                // Find the solving agent
                for agent in agent_query.iter() {
                    if agent.id == *solver_agent_id {
                        // Create new SkillNode
                        let skill_node_mesh = meshes.add(Mesh::from(bevy::render::mesh::shape::Box::new(0.6, 0.6, 0.6)));
                        let skill_material = materials.add(StandardMaterial {
                            base_color: Color::rgb(0.0, 1.0, 0.5),
                            emissive: Color::rgb(0.0, 0.5, 0.25),
                            metallic: 0.8,
                            perceptual_roughness: 0.2,
                            ..default()
                        });

                        // Position near the resolved ErrorNode
                        let skill_position = Vec3::new(
                            fastrand::f32() * 4.0 - 2.0,
                            2.0,
                            fastrand::f32() * 4.0 - 2.0,
                        );

                        let skill_category = match error_node.node_type {
                            ErrorNodeType::LogicError => SkillCategory::Debugging,
                            ErrorNodeType::SecurityVulnerability => SkillCategory::Security,
                            ErrorNodeType::PerformanceBottleneck => SkillCategory::Optimization,
                            ErrorNodeType::IntegrationIssue => SkillCategory::Integration,
                            ErrorNodeType::DataInconsistency => SkillCategory::Analysis,
                            ErrorNodeType::NetworkFailure => SkillCategory::Automation,
                        };

                        let skill_id = format!("skill_{}_{}", error_node.id, fastrand::u32(..));
                        let skill_node = SkillNode {
                            id: skill_id.clone(),
                            name: format!("{:?} Resolution", skill_category),
                            category: skill_category,
                            created_by: agent.owner_id.clone(),
                            usage_count: 0,
                            value: error_node.bounty * 0.5, // Skill value is half the bounty
                            created_at: 0.0, // Would use actual time
                            effectiveness: agent.efficiency,
                        };

                        let entity = commands.spawn((
                            PbrBundle {
                                mesh: skill_node_mesh,
                                material: skill_material,
                                transform: Transform::from_translation(skill_position),
                                ..default()
                            },
                            RigidBody::Fixed,
                            Collider::cuboid(0.3, 0.3, 0.3),
                            skill_node.clone(),
                            TronEffect {
                                glow_intensity: 1.2,
                                pulse_speed: 1.0,
                                color: Color::rgb(0.0, 1.0, 0.5),
                                is_pulsing: true,
                                animation_time: 0.0,
                            },
                            Selectable {
                                is_selected: false,
                                is_hovered: false,
                                selection_radius: 50.0, // Increased for easier selection
                            },
                            Name::new(format!("SkillNode_{}", skill_id)),
                        )).id();

                        knirv_graph.skill_nodes.insert(skill_id.clone(), entity);
                        player_resources.skills_learned += 1;

                        info!("Generated SkillNode {} from resolved ErrorNode {}", skill_id, error_node.id);
                        break;
                    }
                }
            }
        }
    }
}

/// System for collaborative idea node development (different from competitive error solving)
pub fn idea_collaboration_system(
    mut commands: Commands,
    mut meshes: ResMut<Assets<Mesh>>,
    mut materials: ResMut<Assets<StandardMaterial>>,
    mut knirv_graph: ResMut<KnirvGraphState>,
    mut idea_node_query: Query<(Entity, &mut IdeaNode, &Transform)>,
    agent_query: Query<(Entity, &AIAgent, &Transform)>,
    time: Res<Time>,
) {
    for (idea_entity, mut idea_node, idea_transform) in idea_node_query.iter_mut() {
        if idea_node.status == IdeaStatus::Pending {
            // Find nearby agents to collaborate on this idea
            let mut nearby_agents = Vec::new();

            for (agent_entity, agent, agent_transform) in agent_query.iter() {
                // Check if agent is within collaboration range
                let distance = agent_transform.translation.distance(idea_transform.translation);
                if distance < 5.0 && !idea_node.collaborators.contains(&agent.id) {
                    nearby_agents.push(agent.id.clone());
                }
            }

            // Add collaborators and assign stakes
            for agent_id in nearby_agents {
                idea_node.collaborators.push(agent_id.clone());
                // Equal stakes for all collaborators initially
                let stake = 1.0 / (idea_node.collaborators.len() as f32);
                idea_node.stakes.insert(agent_id, stake);
            }

            if !idea_node.collaborators.is_empty() {
                idea_node.status = IdeaStatus::Collaborative;
                info!("IdeaNode {} entered collaborative phase with {} agents", idea_node.id, idea_node.collaborators.len());
            }
        }

        // Progress collaborative development
        if idea_node.status == IdeaStatus::Collaborative {
            let collaboration_progress = idea_node.collaborators.len() as f32 * time.delta_seconds() * 0.1;
            idea_node.feasibility_score = (idea_node.feasibility_score + collaboration_progress).min(1.0);

            // Check if ready to create property
            if idea_node.feasibility_score >= 0.8 && idea_node.collaborators.len() >= 2 {
                // Create PropertyNode
                create_property_from_idea(&mut commands, &mut meshes, &mut materials, &mut knirv_graph, &idea_node);
                idea_node.status = IdeaStatus::PropertyCreated;
            }
        }
    }
}

/// Helper function to create PropertyNode from IdeaNode
fn create_property_from_idea(
    commands: &mut Commands,
    meshes: &mut ResMut<Assets<Mesh>>,
    materials: &mut ResMut<Assets<StandardMaterial>>,
    knirv_graph: &mut ResMut<KnirvGraphState>,
    idea_node: &IdeaNode,
) {
    let property_id = format!("property_{}", idea_node.id);

    let property_node = PropertyNode {
        id: property_id.clone(),
        name: format!("Property from {}", idea_node.name),
        property_type: match idea_node.idea_type {
            IdeaType::Asset => PropertyType::Asset,
            IdeaType::Characteristic => PropertyType::Characteristic,
            IdeaType::Attribute => PropertyType::Attribute,
            IdeaType::Innovation => PropertyType::Patent,
            IdeaType::Improvement => PropertyType::License,
            IdeaType::Feature => PropertyType::Trademark,
        },
        source_idea: idea_node.id.clone(),
        value_type: "object".to_string(),
        immutable: true,
        category: "collaborative".to_string(),
        owners: idea_node.stakes.clone(),
        market_value: idea_node.collaboration_value,
        usage_count: 0,
        created_at: std::time::SystemTime::now().duration_since(std::time::UNIX_EPOCH).unwrap().as_secs_f64(),
    };

    // Create visual representation
    let property_mesh = meshes.add(Mesh::from(bevy::render::mesh::shape::UVSphere { radius: 0.4, sectors: 16, stacks: 8 }));
    let property_material = materials.add(StandardMaterial {
        base_color: Color::rgb(1.0, 0.8, 0.0), // Golden color for properties
        emissive: Color::rgb(0.5, 0.4, 0.0),
        metallic: 0.9,
        perceptual_roughness: 0.1,
        ..default()
    });

    let property_position = Vec3::new(
        (fastrand::f32() - 0.5) * 40.0,
        1.0,
        (fastrand::f32() - 0.5) * 40.0,
    );

    let entity = commands.spawn((
        PbrBundle {
            mesh: property_mesh,
            material: property_material,
            transform: Transform::from_translation(property_position),
            ..default()
        },
        RigidBody::Fixed,
        Collider::ball(0.4),
        property_node.clone(),
        TronEffect {
            glow_intensity: 1.5,
            pulse_speed: 0.8,
            color: Color::rgb(1.0, 0.8, 0.0),
            is_pulsing: true,
            animation_time: 0.0,
        },
        Selectable {
            is_selected: false,
            is_hovered: false,
            selection_radius: 50.0,
        },
        Name::new(format!("PropertyNode_{}", property_id)),
    )).id();

    knirv_graph.property_nodes.insert(property_id.clone(), entity);
    info!("Created PropertyNode {} from collaborative IdeaNode {}", property_id, idea_node.id);
}

/// System to handle competitive resolution when multiple agents work on same ErrorNode
pub fn competitive_resolution_system(
    mut agent_query: Query<&mut AIAgent>,
    mut error_node_query: Query<&mut ErrorNode>,
    mut player_resources: ResMut<PlayerResources>,
    time: Res<Time>,
) {
    // Track which ErrorNodes have multiple agents working on them
    let mut error_node_agents: std::collections::HashMap<String, Vec<String>> = std::collections::HashMap::new();

    // Collect agents working on each ErrorNode
    for agent in agent_query.iter() {
        if let Some(task_id) = &agent.current_task {
            if agent.status == AgentStatus::Working {
                error_node_agents.entry(task_id.clone())
                    .or_insert_with(Vec::new)
                    .push(agent.id.clone());
            }
        }
    }

    // Handle competitive scenarios
    for (error_node_id, agent_ids) in error_node_agents.iter() {
        if agent_ids.len() > 1 {
            // Multiple agents working on same ErrorNode - competitive mode
            for mut error_node in error_node_query.iter_mut() {
                if error_node.id == *error_node_id {
                    // Increase progress rate due to collaboration, but add some randomness for competition
                    let collaboration_bonus = 1.2;
                    let competition_factor = 0.8 + fastrand::f32() * 0.4; // 0.8 to 1.2 random factor

                    // The current solver gets the bonus
                    if let Some(current_solver) = &error_node.solver_agent_id {
                        for mut agent in agent_query.iter_mut() {
                            if agent.id == *current_solver {
                                // Apply competitive bonus to efficiency
                                let temp_efficiency = agent.efficiency * collaboration_bonus * competition_factor;
                                agent.efficiency = temp_efficiency.min(2.5); // Cap at 2.5 for competitive scenarios
                                break;
                            }
                        }
                    }
                    break;
                }
            }
        }
    }
}

// ============================================================================
// KNIRV-GRAPH VISUALIZATION SYSTEMS
// ============================================================================

/// System to create and update dynamic connections between nodes
pub fn update_graph_connections(
    mut commands: Commands,
    mut meshes: ResMut<Assets<Mesh>>,
    mut materials: ResMut<Assets<StandardMaterial>>,
    mut knirv_graph: ResMut<KnirvGraphState>,
    error_node_query: Query<(Entity, &Transform), With<ErrorNode>>,
    skill_node_query: Query<(Entity, &Transform), With<SkillNode>>,
    connection_query: Query<Entity, With<GraphConnection>>,
) {
    // Clear existing connections
    for entity in connection_query.iter() {
        commands.entity(entity).despawn();
    }
    knirv_graph.connections.clear();

    let connection_material = materials.add(StandardMaterial {
        base_color: Color::rgba(0.0, 0.8, 1.0, 0.6),
        emissive: Color::rgb(0.0, 0.4, 0.5),
        unlit: true,
        alpha_mode: AlphaMode::Blend,
        ..default()
    });

    // Create connections between nearby nodes
    let mut error_positions: Vec<(Entity, Vec3)> = error_node_query.iter()
        .map(|(entity, transform)| (entity, transform.translation))
        .collect();

    let skill_positions: Vec<(Entity, Vec3)> = skill_node_query.iter()
        .map(|(entity, transform)| (entity, transform.translation))
        .collect();

    // Connect ErrorNodes to nearby SkillNodes
    for (error_entity, error_pos) in error_positions.iter() {
        for (skill_entity, skill_pos) in skill_positions.iter() {
            let distance = error_pos.distance(*skill_pos);
            if distance < 20.0 { // Connect if within 20 units
                create_connection_line(
                    &mut commands,
                    &mut meshes,
                    connection_material.clone(),
                    *error_pos,
                    *skill_pos,
                    *error_entity,
                    *skill_entity,
                    ConnectionType::SkillDependency,
                    &mut knirv_graph,
                );
            }
        }
    }

    // Connect ErrorNodes to other nearby ErrorNodes
    for i in 0..error_positions.len() {
        for j in (i + 1)..error_positions.len() {
            let (entity1, pos1) = error_positions[i];
            let (entity2, pos2) = error_positions[j];
            let distance = pos1.distance(pos2);

            if distance < 15.0 { // Connect if within 15 units
                create_connection_line(
                    &mut commands,
                    &mut meshes,
                    connection_material.clone(),
                    pos1,
                    pos2,
                    entity1,
                    entity2,
                    ConnectionType::ErrorPropagation,
                    &mut knirv_graph,
                );
            }
        }
    }
}

/// Helper function to create a connection line between two nodes
fn create_connection_line(
    commands: &mut Commands,
    meshes: &mut ResMut<Assets<Mesh>>,
    material: Handle<StandardMaterial>,
    start_pos: Vec3,
    end_pos: Vec3,
    start_entity: Entity,
    end_entity: Entity,
    connection_type: ConnectionType,
    knirv_graph: &mut ResMut<KnirvGraphState>,
) {
    let direction = (end_pos - start_pos).normalize();
    let distance = start_pos.distance(end_pos);
    let midpoint = (start_pos + end_pos) / 2.0;

    // Create a thin cylinder as the connection line
    let line_mesh = meshes.add(Mesh::from(bevy::render::mesh::shape::Cylinder {
        radius: 0.02,
        height: distance,
        resolution: 8,
        segments: 1,
    }));

    // Calculate rotation to align with direction
    let up = Vec3::Y;
    let rotation = if direction.dot(up).abs() > 0.99 {
        Quat::IDENTITY
    } else {
        Quat::from_rotation_arc(up, direction)
    };

    let connection_entity = commands.spawn((
        PbrBundle {
            mesh: line_mesh,
            material,
            transform: Transform {
                translation: midpoint,
                rotation,
                scale: Vec3::ONE,
            },
            ..default()
        },
        GraphConnection {
            from_node: start_entity,
            to_node: end_entity,
            connection_type,
            data_flow_speed: 2.0,
            is_active: true,
        },
        Name::new("Graph Connection"),
    )).id();

    knirv_graph.connections.push((start_entity, end_entity));
}

/// System to animate data flow along connections
pub fn animate_data_flow(
    mut commands: Commands,
    mut meshes: ResMut<Assets<Mesh>>,
    mut materials: ResMut<Assets<StandardMaterial>>,
    connection_query: Query<(&GraphConnection, &Transform)>,
    time: Res<Time>,
) {
    let particle_material = materials.add(StandardMaterial {
        base_color: Color::rgb(0.0, 1.0, 0.8),
        emissive: Color::rgb(0.0, 0.5, 0.4),
        unlit: true,
        ..default()
    });

    for (connection, transform) in connection_query.iter() {
        if connection.is_active && fastrand::f32() < 0.1 { // 10% chance to spawn particle
            let particle_mesh = meshes.add(Mesh::from(bevy::render::mesh::shape::UVSphere {
                radius: 0.05,
                sectors: 8,
                stacks: 4,
            }));

            // Spawn data flow particle
            commands.spawn((
                PbrBundle {
                    mesh: particle_mesh,
                    material: particle_material.clone(),
                    transform: Transform::from_translation(transform.translation),
                    ..default()
                },
                DataFlowParticle {
                    speed: connection.data_flow_speed,
                    lifetime: 3.0,
                    age: 0.0,
                },
                Name::new("Data Flow Particle"),
            ));
        }
    }
}

/// Component for data flow particles
#[derive(Component)]
pub struct DataFlowParticle {
    pub speed: f32,
    pub lifetime: f32,
    pub age: f32,
}

/// System to update data flow particles
pub fn update_data_flow_particles(
    mut commands: Commands,
    mut particle_query: Query<(Entity, &mut Transform, &mut DataFlowParticle)>,
    time: Res<Time>,
) {
    for (entity, mut transform, mut particle) in particle_query.iter_mut() {
        particle.age += time.delta_seconds();

        if particle.age >= particle.lifetime {
            commands.entity(entity).despawn();
            continue;
        }

        // Move particle along a random path
        let movement = Vec3::new(
            (particle.age * particle.speed).sin() * 0.1,
            particle.speed * time.delta_seconds(),
            (particle.age * particle.speed * 1.3).cos() * 0.1,
        );

        transform.translation += movement;

        // Fade out over time
        let alpha = 1.0 - (particle.age / particle.lifetime);
        transform.scale = Vec3::splat(alpha);
    }
}

/// System to update graph center and radius based on node positions
pub fn update_graph_bounds(
    mut knirv_graph: ResMut<KnirvGraphState>,
    error_node_query: Query<&Transform, With<ErrorNode>>,
    skill_node_query: Query<&Transform, (With<SkillNode>, Without<ErrorNode>)>,
) {
    let mut positions = Vec::new();

    // Collect all node positions
    for transform in error_node_query.iter() {
        positions.push(transform.translation);
    }

    for transform in skill_node_query.iter() {
        positions.push(transform.translation);
    }

    if positions.is_empty() {
        return;
    }

    // Calculate center
    let center = positions.iter().fold(Vec3::ZERO, |acc, pos| acc + *pos) / positions.len() as f32;
    knirv_graph.graph_center = center;

    // Calculate radius (maximum distance from center)
    let max_distance = positions.iter()
        .map(|pos| center.distance(*pos))
        .fold(0.0, f32::max);

    knirv_graph.graph_radius = max_distance + 5.0; // Add some padding
}

// ============================================================================
// MULTIPLAYER COMPETITIVE MECHANICS
// ============================================================================

/// System to handle multiplayer competitive scenarios
pub fn multiplayer_competition_system(
    mut competitive_state: ResMut<CompetitiveState>,
    mut agent_query: Query<&mut AIAgent>,
    mut error_node_query: Query<&mut ErrorNode>,
    mut player_resources: ResMut<PlayerResources>,
    time: Res<Time>,
) {
    let current_time = time.elapsed_seconds_f64();

    // Update active competitions
    for competition in competitive_state.current_competitions.iter_mut() {
        // Check if competition is still active
        if current_time > competition.estimated_completion {
            // Determine winner based on progress
            let mut max_progress = 0.0;
            let mut winner = None;

            for (player_id, progress) in competition.progress.iter() {
                if *progress > max_progress {
                    max_progress = *progress;
                    winner = Some(player_id.clone());
                }
            }

            if let Some(winner_id) = winner {
                info!("Competition for ErrorNode {} won by player {}",
                      competition.error_node_id, winner_id);

                // Award bonus for winning competition
                if winner_id == "player_1" { // Current player
                    player_resources.nrn_balance += 50.0; // Competition bonus
                    player_resources.current_streak += 1;
                }
            }
        }
    }

    // Remove completed competitions
    competitive_state.current_competitions.retain(|comp|
        current_time <= comp.estimated_completion
    );

    // Create new competitions for ErrorNodes with multiple agents
    let mut error_node_agents: std::collections::HashMap<String, Vec<String>> =
        std::collections::HashMap::new();

    for agent in agent_query.iter() {
        if let Some(task_id) = &agent.current_task {
            if agent.status == AgentStatus::Working {
                error_node_agents.entry(task_id.clone())
                    .or_insert_with(Vec::new)
                    .push(agent.owner_id.clone());
            }
        }
    }

    // Start new competitions for nodes with multiple players
    for (error_node_id, player_ids) in error_node_agents.iter() {
        let unique_players: std::collections::HashSet<_> = player_ids.iter().collect();

        let unique_player_count = unique_players.len();
        if unique_player_count > 1 {
            // Check if competition already exists
            let competition_exists = competitive_state.current_competitions.iter()
                .any(|comp| comp.error_node_id == *error_node_id);

            if !competition_exists {
                // Create new competition
                let mut progress = std::collections::HashMap::new();
                for player_id in &unique_players {
                    progress.insert((*player_id).clone(), 0.0);
                }

                let competition = CompetitionInfo {
                    error_node_id: error_node_id.clone(),
                    participants: player_ids.clone(),
                    progress,
                    started_at: current_time,
                    estimated_completion: current_time + 60.0, // 1 minute competition
                };

                competitive_state.current_competitions.push(competition);
                info!("Started competition for ErrorNode {} with {} players",
                      error_node_id, unique_player_count);
            }
        }
    }
}

/// System to update leaderboard based on player performance
pub fn update_leaderboard_system(
    mut competitive_state: ResMut<CompetitiveState>,
    player_resources: Res<PlayerResources>,
    time: Res<Time>,
) {
    let current_time = time.elapsed_seconds_f64();

    // Update leaderboard periodically
    if current_time - competitive_state.last_leaderboard_update > 10.0 {
        competitive_state.last_leaderboard_update = current_time;

        // Update current player's entry
        let player_entry = LeaderboardEntry {
            player_id: "player_1".to_string(),
            player_name: "You".to_string(),
            score: player_resources.errors_resolved * 100 + player_resources.skills_learned * 50,
            errors_resolved: player_resources.errors_resolved,
            nrn_earned: player_resources.total_bounty_earned,
        };

        // Update or add player entry
        if let Some(existing_entry) = competitive_state.leaderboard.iter_mut()
            .find(|entry| entry.player_id == "player_1") {
            *existing_entry = player_entry;
        } else {
            competitive_state.leaderboard.push(player_entry);
        }

        // Sort leaderboard by score
        competitive_state.leaderboard.sort_by(|a, b| b.score.cmp(&a.score));

        // Keep only top 10
        competitive_state.leaderboard.truncate(10);

        // Add some simulated other players for demonstration
        if competitive_state.leaderboard.len() < 5 {
            for i in competitive_state.leaderboard.len()..5 {
                let simulated_entry = LeaderboardEntry {
                    player_id: format!("bot_{}", i),
                    player_name: format!("Player {}", i + 1),
                    score: fastrand::u32(50..500),
                    errors_resolved: fastrand::u32(1..20),
                    nrn_earned: fastrand::f64() * 1000.0,
                };
                competitive_state.leaderboard.push(simulated_entry);
            }

            // Re-sort after adding simulated players
            competitive_state.leaderboard.sort_by(|a, b| b.score.cmp(&a.score));
        }
    }
}

/// System to synchronize game state across multiplayer sessions
pub fn multiplayer_sync_system(
    competitive_state: Res<CompetitiveState>,
    player_resources: Res<PlayerResources>,
    mut network_manager: ResMut<NetworkManager>,
    time: Res<Time>,
) {
    // Simulate network synchronization
    if time.elapsed_seconds_f64() - network_manager.last_ping > 5.0 {
        network_manager.last_ping = time.elapsed_seconds_f64();

        // Simulate sending player state to server
        let player_state = serde_json::json!({
            "player_id": "player_1",
            "nrn_balance": player_resources.nrn_balance,
            "errors_resolved": player_resources.errors_resolved,
            "skills_learned": player_resources.skills_learned,
            "timestamp": time.elapsed_seconds_f64()
        });

        // In a real implementation, this would send data to the server
        debug!("Syncing player state: {}", player_state);

        // Simulate receiving updates from other players
        if fastrand::f32() < 0.3 { // 30% chance to receive update
            // This would normally come from the server
            info!("Received multiplayer state update");
        }
    }
}

/// System to handle real-time progress comparison
pub fn progress_comparison_system(
    competitive_state: Res<CompetitiveState>,
    error_node_query: Query<&ErrorNode>,
    mut commands: Commands,
    asset_server: Res<AssetServer>,
) {
    // Display progress comparison for active competitions
    for competition in competitive_state.current_competitions.iter() {
        // Find the ErrorNode being competed for
        for error_node in error_node_query.iter() {
            if error_node.id == competition.error_node_id {
                // Create visual indicators for competitive progress
                // This would spawn UI elements showing other players' progress

                let progress_text = format!(
                    "Competition: {} players working on {}",
                    competition.participants.len(),
                    error_node.id
                );

                // In a real implementation, this would create floating UI elements
                debug!("{}", progress_text);
                break;
            }
        }
    }
}
