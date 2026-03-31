use bevy::prelude::*;
use crate::components::*;
use crate::resources::*;

pub struct MobilePlugin;

impl Plugin for MobilePlugin {
    fn build(&self, app: &mut App) {
        app
            .init_resource::<MobileSettings>()
            .init_resource::<TouchControlState>()
            .add_systems(Startup, setup_mobile_optimizations)
            .add_systems(Update, (
                mobile_performance_monitor,
                adaptive_quality_system,
                battery_optimization_system,
                knirvana_touch_controls,
                mobile_ui_optimization,
                mobile_graphics_optimization,
            ));
    }
}

/// Resource for managing touch control state
#[derive(Resource, Default)]
pub struct TouchControlState {
    pub selected_agent: Option<Entity>,
    pub selected_error_node: Option<Entity>,
    pub touch_start_time: f64,
    pub last_tap_position: Option<Vec2>,
    pub double_tap_threshold: f64,
    pub drag_threshold: f32,
    pub is_dragging: bool,
}

fn setup_mobile_optimizations(
    mut commands: Commands,
    mut mobile_settings: ResMut<MobileSettings>,
) {
    #[cfg(feature = "mobile")]
    {
        info!("Setting up mobile optimizations");

        mobile_settings.touch_sensitivity = 1.0;
        mobile_settings.graphics_quality = GraphicsQuality::Medium;
        mobile_settings.battery_optimization = true;
        mobile_settings.reduced_effects = true;

        // Apply mobile-specific entity optimizations
        commands.spawn((
            MobileOptimized {
                lod_level: 2,
                simplified_physics: true,
                reduced_particles: true,
            },
            Name::new("Mobile Optimization Manager"),
        ));
    }

    #[cfg(not(feature = "mobile"))]
    {
        mobile_settings.graphics_quality = GraphicsQuality::High;
        mobile_settings.battery_optimization = false;
        mobile_settings.reduced_effects = false;
    }
}

fn mobile_performance_monitor(
    time: Res<Time>,
    mut mobile_settings: ResMut<MobileSettings>,
    mut metrics: ResMut<GameMetrics>,
) {
    #[cfg(feature = "mobile")]
    {
        metrics.frame_rate = 1.0 / time.delta_seconds();

        // Adaptive quality based on performance
        if metrics.frame_rate < 30.0 {
            match mobile_settings.graphics_quality {
                GraphicsQuality::High => {
                    mobile_settings.graphics_quality = GraphicsQuality::Medium;
                    info!("Reduced graphics quality to Medium due to low FPS");
                }
                GraphicsQuality::Medium => {
                    mobile_settings.graphics_quality = GraphicsQuality::Low;
                    info!("Reduced graphics quality to Low due to low FPS");
                }
                _ => {}
            }
        } else if metrics.frame_rate > 55.0 {
            match mobile_settings.graphics_quality {
                GraphicsQuality::Low => {
                    mobile_settings.graphics_quality = GraphicsQuality::Medium;
                    info!("Increased graphics quality to Medium");
                }
                GraphicsQuality::Medium => {
                    mobile_settings.graphics_quality = GraphicsQuality::High;
                    info!("Increased graphics quality to High");
                }
                _ => {}
            }
        }
    }
}

fn adaptive_quality_system(
    mobile_settings: Res<MobileSettings>,
    mut mobile_optimized_query: Query<&mut MobileOptimized>,
) {
    #[cfg(feature = "mobile")]
    {
        if mobile_settings.is_changed() {
            for mut optimized in mobile_optimized_query.iter_mut() {
                match mobile_settings.graphics_quality {
                    GraphicsQuality::Low => {
                        optimized.lod_level = 3;
                        optimized.simplified_physics = true;
                        optimized.reduced_particles = true;
                    }
                    GraphicsQuality::Medium => {
                        optimized.lod_level = 2;
                        optimized.simplified_physics = true;
                        optimized.reduced_particles = true;
                    }
                    GraphicsQuality::High => {
                        optimized.lod_level = 1;
                        optimized.simplified_physics = false;
                        optimized.reduced_particles = false;
                    }
                    GraphicsQuality::Ultra => {
                        optimized.lod_level = 0;
                        optimized.simplified_physics = false;
                        optimized.reduced_particles = false;
                    }
                }
            }
        }
    }
}

fn battery_optimization_system(
    mobile_settings: Res<MobileSettings>,
    time: Res<Time>,
) {
    #[cfg(feature = "mobile")]
    {
        if mobile_settings.battery_optimization {
            // Reduce update frequency for non-critical systems
            let frame_skip = (time.elapsed_seconds() * 60.0) as u32 % 2;

            if frame_skip == 0 {
                // Skip some updates to save battery
                return;
            }
        }
    }
}

// Mobile-specific input handling
pub fn handle_mobile_gestures(
    touches: Res<Touches>,
    mut player_query: Query<&mut Transform, With<PlayerAvatar>>,
    mobile_settings: Res<MobileSettings>,
) {
    #[cfg(feature = "mobile")]
    {
        // Handle pinch-to-zoom
        if touches.iter().count() == 2 {
            let touches: Vec<_> = touches.iter().collect();
            if touches.len() == 2 {
                let touch1 = touches[0].position();
                let touch2 = touches[1].position();
                let distance = touch1.distance(touch2);

                // Implement zoom logic based on pinch distance
                info!("Pinch gesture detected, distance: {}", distance);
            }
        }

        // Handle swipe gestures
        for touch in touches.iter() {
            if let Some(previous_position) = touches.get_pressed(touch.id()) {
                let current_position = touch.position();
                let delta = current_position - previous_position;

                if delta.length() > 100.0 * mobile_settings.touch_sensitivity {
                    info!("Swipe gesture detected: {:?}", delta);
                }
            }
        }
    }
}

// Platform-specific initialization
#[cfg(target_os = "android")]
pub fn android_init() {
    use android_logger::{Config, FilterBuilder};

    android_logger::init_once(
        Config::default()
            .with_min_level(log::Level::Info)
            .with_tag("KNIRVANA")
            .with_filter(FilterBuilder::new().parse("debug,hello::crate=trace").build())
    );

    info!("Android platform initialized");
}

#[cfg(target_os = "ios")]
pub fn ios_init() {
    info!("iOS platform initialized");
}

// ============================================================================
// KNIRVANA-SPECIFIC MOBILE OPTIMIZATIONS
// ============================================================================

/// System for KNIRVANA-specific touch controls
pub fn knirvana_touch_controls(
    touches: Res<Touches>,
    mut touch_state: ResMut<TouchControlState>,
    mut game_state: ResMut<KnirvanaGameState>,
    mut player_resources: ResMut<PlayerResources>,
    mut agent_query: Query<(Entity, &mut AIAgent, &Transform), With<AIAgent>>,
    mut error_node_query: Query<(Entity, &mut ErrorNode, &Transform), With<ErrorNode>>,
    camera_query: Query<(&Camera, &GlobalTransform), With<Camera3d>>,
    time: Res<Time>,
    mobile_settings: Res<MobileSettings>,
) {
    #[cfg(feature = "mobile")]
    {
        let current_time = time.elapsed_seconds_f64();

        // Handle touch input for agent deployment
        for touch in touches.iter() {
            let touch_position = touch.position();

            if touches.just_pressed(touch.id()) {
                touch_state.touch_start_time = current_time;
                touch_state.last_tap_position = Some(touch_position);
                touch_state.is_dragging = false;

                // Raycast from touch position to 3D world
                if let Ok((camera, camera_transform)) = camera_query.get_single() {
                    if let Some(ray) = camera.viewport_to_world(camera_transform, touch_position) {
                        // Check for agent selection
                        for (entity, agent, transform) in agent_query.iter() {
                            let distance = ray.origin.distance(transform.translation);
                            if distance < 2.0 { // Within selection range
                                touch_state.selected_agent = Some(entity);
                                game_state.selected_agent = Some(entity);
                                info!("Selected agent: {}", agent.id);
                                break;
                            }
                        }

                        // Check for ErrorNode selection
                        for (entity, error_node, transform) in error_node_query.iter() {
                            let distance = ray.origin.distance(transform.translation);
                            if distance < 2.0 { // Within selection range
                                touch_state.selected_error_node = Some(entity);
                                game_state.selected_error_node = Some(entity);
                                info!("Selected ErrorNode: {}", error_node.id);
                                break;
                            }
                        }
                    }
                }
            }

            if touches.just_released(touch.id()) {
                let touch_duration = current_time - touch_state.touch_start_time;

                // Handle double-tap for agent deployment
                if touch_duration < 0.3 && !touch_state.is_dragging {
                    if let Some(last_pos) = touch_state.last_tap_position {
                        if touch_position.distance(last_pos) < 50.0 {
                            // Double tap detected - deploy agent
                            if let (Some(agent_entity), Some(error_node_entity)) =
                                (touch_state.selected_agent, touch_state.selected_error_node) {

                                // Deploy agent to ErrorNode
                                if let Ok((_, mut agent, _)) = agent_query.get_mut(agent_entity) {
                                    if let Ok((_, mut error_node, _)) = error_node_query.get_mut(error_node_entity) {
                                        let deployment_cost = 10.0;
                                        if player_resources.nrn_balance >= deployment_cost {
                                            player_resources.nrn_balance -= deployment_cost;
                                            agent.current_task = Some(error_node.id.clone());
                                            agent.status = AgentStatus::Moving;
                                            error_node.is_being_solved = true;
                                            error_node.solver_agent_id = Some(agent.id.clone());

                                            info!("Mobile deployment: Agent {} -> ErrorNode {}",
                                                  agent.id, error_node.id);
                                        }
                                    }
                                }
                            }
                        }
                    }
                }

                touch_state.is_dragging = false;
            }

            // Handle drag gestures for camera movement
            if let Some(start_pos) = touch_state.last_tap_position {
                let drag_distance = touch_position.distance(start_pos);
                if drag_distance > touch_state.drag_threshold * mobile_settings.touch_sensitivity {
                    touch_state.is_dragging = true;
                    // Camera movement would be handled here
                }
            }
        }
    }
}

/// System to optimize UI for mobile devices
pub fn mobile_ui_optimization(
    mobile_settings: Res<MobileSettings>,
    mut ui_state: ResMut<UIState>,
    mut style_query: Query<&mut Style>,
    mut text_query: Query<&mut Text>,
) {
    #[cfg(feature = "mobile")]
    {
        if mobile_settings.is_changed() {
            // Adjust UI scale based on graphics quality
            let ui_scale = match mobile_settings.graphics_quality {
                GraphicsQuality::Low => 1.2,    // Larger UI for low-end devices
                GraphicsQuality::Medium => 1.0,
                GraphicsQuality::High => 0.9,
                GraphicsQuality::Ultra => 0.8,
            };

            // Adjust font sizes for mobile readability
            for mut text in text_query.iter_mut() {
                for section in text.sections.iter_mut() {
                    section.style.font_size *= ui_scale;
                }
            }

            // Simplify UI for battery optimization
            if mobile_settings.battery_optimization {
                ui_state.hud_opacity = 0.8; // Reduce opacity to save power
                ui_state.show_leaderboard = false; // Hide complex UI elements
            }
        }
    }
}

/// System to optimize graphics for mobile devices
pub fn mobile_graphics_optimization(
    mobile_settings: Res<MobileSettings>,
    mut visual_effects: ResMut<VisualEffectsState>,
    mut tron_effect_query: Query<&mut TronEffect>,
    mut materials: ResMut<Assets<StandardMaterial>>,
) {
    #[cfg(feature = "mobile")]
    {
        if mobile_settings.is_changed() {
            match mobile_settings.graphics_quality {
                GraphicsQuality::Low => {
                    // Reduce visual effects for low-end devices
                    visual_effects.tron_glow_intensity = 0.5;
                    visual_effects.particle_density = 0.3;
                    visual_effects.bloom_intensity = 0.2;
                    visual_effects.screen_effects_enabled = false;

                    // Reduce TronEffect intensity
                    for mut tron_effect in tron_effect_query.iter_mut() {
                        tron_effect.glow_intensity *= 0.5;
                        tron_effect.is_pulsing = false; // Disable pulsing to save performance
                    }
                },
                GraphicsQuality::Medium => {
                    visual_effects.tron_glow_intensity = 0.8;
                    visual_effects.particle_density = 0.6;
                    visual_effects.bloom_intensity = 0.5;
                    visual_effects.screen_effects_enabled = true;
                },
                GraphicsQuality::High | GraphicsQuality::Ultra => {
                    visual_effects.tron_glow_intensity = 1.0;
                    visual_effects.particle_density = 1.0;
                    visual_effects.bloom_intensity = 0.8;
                    visual_effects.screen_effects_enabled = true;
                },
            }

            // Apply battery optimization
            if mobile_settings.battery_optimization {
                visual_effects.connection_animation_speed *= 0.5; // Slower animations
                visual_effects.ambient_pulse_speed *= 0.7;
            }
        }
    }
}

/// System to handle mobile-specific agent AI optimizations
pub fn mobile_agent_optimization(
    mobile_settings: Res<MobileSettings>,
    mut agent_query: Query<&mut AIAgent>,
    time: Res<Time>,
) {
    #[cfg(feature = "mobile")]
    {
        if mobile_settings.battery_optimization {
            // Reduce AI update frequency on mobile
            let frame_skip = (time.elapsed_seconds() * 30.0) as u32 % 3; // Update every 3rd frame

            if frame_skip != 0 {
                return;
            }

            // Simplify agent thought processes for mobile
            for mut agent in agent_query.iter_mut() {
                if agent.thought_process.len() > 3 {
                    agent.thought_process.truncate(3); // Keep only 3 thoughts on mobile
                }
            }
        }
    }
}

/// System to handle mobile device orientation changes
pub fn handle_orientation_change(
    mut windows: Query<&mut Window>,
    mut ui_state: ResMut<UIState>,
) {
    #[cfg(feature = "mobile")]
    {
        for window in windows.iter() {
            let aspect_ratio = window.width() / window.height();

            // Adjust UI layout based on orientation
            if aspect_ratio > 1.5 {
                // Landscape mode - show more UI elements
                ui_state.show_agent_panel = true;
                ui_state.show_node_info = true;
            } else {
                // Portrait mode - minimize UI for better gameplay
                ui_state.show_agent_panel = false;
                ui_state.show_node_info = false;
            }
        }
    }
}

/// Mobile-specific haptic feedback system
pub fn mobile_haptic_feedback(
    agent_query: Query<&AIAgent, Changed<AIAgent>>,
    error_node_query: Query<&ErrorNode, Changed<ErrorNode>>,
) {
    #[cfg(feature = "mobile")]
    {
        // Provide haptic feedback for important game events
        for agent in agent_query.iter() {
            match agent.status {
                AgentStatus::Working => {
                    // Light haptic feedback when agent starts working
                    info!("Haptic: Agent started working");
                },
                AgentStatus::Idle => {
                    // Haptic feedback when agent completes task
                    info!("Haptic: Agent completed task");
                },
                _ => {}
            }
        }

        for error_node in error_node_query.iter() {
            if error_node.progress >= 1.0 {
                // Strong haptic feedback when ErrorNode is solved
                info!("Haptic: ErrorNode solved!");
            }
        }
    }
}
