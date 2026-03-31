use bevy::prelude::*;
use crate::components::*;
use crate::resources::*;

// ============================================================================
// KNIRVANA UI COMPONENTS
// ============================================================================

#[derive(Component)]
pub struct GameHUD;

#[derive(Component)]
pub struct NRNDisplay;

#[derive(Component)]
pub struct AgentPanel;

#[derive(Component)]
pub struct ErrorNodeInfo;

#[derive(Component)]
pub struct SkillNodeInfo;

#[derive(Component)]
pub struct ThoughtDisplay;

#[derive(Component)]
pub struct ProgressBar;

#[derive(Component)]
pub struct LeaderboardPanel;

#[derive(Component)]
pub struct NotificationPanel;

// ============================================================================
// KNIRVANA UI SETUP SYSTEMS
// ============================================================================

/// Setup the main game HUD
pub fn setup_knirvana_ui(
    mut commands: Commands,
    asset_server: Res<AssetServer>,
) {
    info!("Setting up KNIRVANA UI");

    // Main UI root
    commands.spawn((
        NodeBundle {
            style: Style {
                width: Val::Percent(100.0),
                height: Val::Percent(100.0),
                position_type: PositionType::Absolute,
                ..default()
            },
            background_color: Color::NONE.into(),
            ..default()
        },
        Name::new("KNIRVANA UI Root"),
    )).with_children(|parent| {
        // Top HUD bar
        setup_top_hud(parent, &asset_server);
        
        // Left panel for agent controls
        setup_agent_panel(parent, &asset_server);
        
        // Right panel for node information
        setup_info_panel(parent, &asset_server);
        
        // Bottom notification area
        setup_notification_panel(parent, &asset_server);
        
        // Leaderboard overlay (initially hidden)
        setup_leaderboard_panel(parent, &asset_server);
    });
}

/// Setup the top HUD showing resources and stats
fn setup_top_hud(parent: &mut ChildBuilder, asset_server: &Res<AssetServer>) {
    parent.spawn((
        NodeBundle {
            style: Style {
                width: Val::Percent(100.0),
                height: Val::Px(80.0),
                position_type: PositionType::Absolute,
                top: Val::Px(0.0),
                padding: UiRect::all(Val::Px(10.0)),
                justify_content: JustifyContent::SpaceBetween,
                align_items: AlignItems::Center,
                ..default()
            },
            background_color: Color::rgba(0.0, 0.1, 0.2, 0.8).into(),
            ..default()
        },
        GameHUD,
        Name::new("Top HUD"),
    )).with_children(|hud| {
        // NRN Balance display
        hud.spawn((
            TextBundle::from_section(
                "NRN: 500.0",
                TextStyle {
                    font: Default::default(), // Use Bevy's default font
                    font_size: 24.0,
                    color: Color::rgb(0.0, 1.0, 0.8),
                },
            ),
            NRNDisplay,
            Name::new("NRN Display"),
        ));

        // Game stats
        hud.spawn((
            TextBundle::from_section(
                "Errors Resolved: 0 | Skills Learned: 0",
                TextStyle {
                    font: Default::default(), // Use Bevy's default font
                    font_size: 18.0,
                    color: Color::rgb(0.8, 0.8, 1.0),
                },
            ),
            Name::new("Game Stats"),
        ));

        // Game phase indicator
        hud.spawn((
            TextBundle::from_section(
                "KNIRVANA - Ready to Play! Press SPACE to start",
                TextStyle {
                    font: Default::default(), // Use Bevy's default font
                    font_size: 20.0,
                    color: Color::rgb(1.0, 1.0, 0.2),
                },
            ),
            Name::new("Game Phase"),
        ));
    });
}

/// Setup the left agent control panel
fn setup_agent_panel(parent: &mut ChildBuilder, asset_server: &Res<AssetServer>) {
    parent.spawn((
        NodeBundle {
            style: Style {
                width: Val::Px(300.0),
                height: Val::Percent(70.0),
                position_type: PositionType::Absolute,
                left: Val::Px(10.0),
                top: Val::Px(90.0),
                padding: UiRect::all(Val::Px(15.0)),
                flex_direction: FlexDirection::Column,
                ..default()
            },
            background_color: Color::rgba(0.1, 0.0, 0.2, 0.9).into(),
            ..default()
        },
        AgentPanel,
        Name::new("Agent Panel"),
    )).with_children(|panel| {
        // Panel title
        panel.spawn(TextBundle::from_section(
            "AI AGENTS",
            TextStyle {
                font: Default::default(), // Use Bevy's default font
                font_size: 20.0,
                color: Color::rgb(0.2, 0.8, 1.0),
            },
        ));
        
        // Agent list container
        panel.spawn((
            NodeBundle {
                style: Style {
                    width: Val::Percent(100.0),
                    height: Val::Percent(80.0),
                    flex_direction: FlexDirection::Column,
                    margin: UiRect::top(Val::Px(10.0)),
                    ..default()
                },
                background_color: Color::rgba(0.0, 0.0, 0.1, 0.5).into(),
                ..default()
            },
            Name::new("Agent List"),
        )).with_children(|agent_list| {
            agent_list.spawn(TextBundle::from_section(
                "Loading agents...",
                TextStyle {
                    font: Default::default(),
                    font_size: 14.0,
                    color: Color::rgb(0.8, 0.8, 0.8),
                },
            ));
        });
        
        // Agent controls
        panel.spawn((
            ButtonBundle {
                style: Style {
                    width: Val::Percent(100.0),
                    height: Val::Px(40.0),
                    margin: UiRect::top(Val::Px(10.0)),
                    justify_content: JustifyContent::Center,
                    align_items: AlignItems::Center,
                    ..default()
                },
                background_color: Color::rgba(0.0, 0.5, 0.8, 0.8).into(),
                ..default()
            },
            Name::new("Deploy Agent Button"),
        )).with_children(|button| {
            button.spawn(TextBundle::from_section(
                "DEPLOY AGENT [SPACE]",
                TextStyle {
                    font: Default::default(), // Use Bevy's default font
                    font_size: 16.0,
                    color: Color::WHITE,
                },
            ));
        });
    });
}

/// Setup the right information panel
fn setup_info_panel(parent: &mut ChildBuilder, asset_server: &Res<AssetServer>) {
    parent.spawn((
        NodeBundle {
            style: Style {
                width: Val::Px(350.0),
                height: Val::Percent(70.0),
                position_type: PositionType::Absolute,
                right: Val::Px(10.0),
                top: Val::Px(90.0),
                padding: UiRect::all(Val::Px(15.0)),
                flex_direction: FlexDirection::Column,
                ..default()
            },
            background_color: Color::rgba(0.2, 0.0, 0.1, 0.9).into(),
            ..default()
        },
        ErrorNodeInfo,
        Name::new("Info Panel"),
    )).with_children(|panel| {
        // Panel title
        panel.spawn(TextBundle::from_section(
            "NODE INFORMATION",
            TextStyle {
                font: Default::default(), // Use Bevy's default font
                font_size: 20.0,
                color: Color::rgb(1.0, 0.2, 0.2),
            },
        ));
        
        // Node details container
        panel.spawn((
            NodeBundle {
                style: Style {
                    width: Val::Percent(100.0),
                    height: Val::Percent(60.0),
                    flex_direction: FlexDirection::Column,
                    margin: UiRect::top(Val::Px(10.0)),
                    padding: UiRect::all(Val::Px(10.0)),
                    ..default()
                },
                background_color: Color::rgba(0.1, 0.0, 0.0, 0.5).into(),
                ..default()
            },
            ErrorNodeInfo,
            Name::new("Node Details"),
        )).with_children(|node_info| {
            node_info.spawn(TextBundle::from_section(
                "Click on an error node to see details here.\n\nControls:\n- Left Click: Select node/agent\n- SPACE: Deploy agent\n- TAB: Toggle leaderboard\n- A: Toggle agent panel\n- I: Toggle info panel",
                TextStyle {
                    font: Default::default(),
                    font_size: 14.0,
                    color: Color::rgb(0.8, 0.8, 0.8),
                },
            ));
        });
        
        // Thought process display
        panel.spawn((
            NodeBundle {
                style: Style {
                    width: Val::Percent(100.0),
                    height: Val::Percent(30.0),
                    flex_direction: FlexDirection::Column,
                    margin: UiRect::top(Val::Px(10.0)),
                    padding: UiRect::all(Val::Px(10.0)),
                    ..default()
                },
                background_color: Color::rgba(0.0, 0.1, 0.0, 0.5).into(),
                ..default()
            },
            ThoughtDisplay,
            Name::new("Thought Process"),
        )).with_children(|thoughts| {
            thoughts.spawn(TextBundle::from_section(
                "AGENT THOUGHTS",
                TextStyle {
                    font: Default::default(), // Use Bevy's default font
                    font_size: 16.0,
                    color: Color::rgb(0.0, 1.0, 0.5),
                },
            ));
        });
    });
}

/// Setup the notification panel at the bottom
fn setup_notification_panel(parent: &mut ChildBuilder, asset_server: &Res<AssetServer>) {
    parent.spawn((
        NodeBundle {
            style: Style {
                width: Val::Percent(100.0),
                height: Val::Px(100.0),
                position_type: PositionType::Absolute,
                bottom: Val::Px(0.0),
                padding: UiRect::all(Val::Px(10.0)),
                justify_content: JustifyContent::Center,
                align_items: AlignItems::FlexEnd,
                ..default()
            },
            background_color: Color::rgba(0.0, 0.0, 0.0, 0.6).into(),
            ..default()
        },
        NotificationPanel,
        Name::new("Notification Panel"),
    ));
}

/// Setup the leaderboard overlay
fn setup_leaderboard_panel(parent: &mut ChildBuilder, asset_server: &Res<AssetServer>) {
    parent.spawn((
        NodeBundle {
            style: Style {
                width: Val::Px(400.0),
                height: Val::Px(500.0),
                position_type: PositionType::Absolute,
                left: Val::Percent(50.0),
                top: Val::Percent(50.0),
                margin: UiRect {
                    left: Val::Px(-200.0), // Center horizontally
                    top: Val::Px(-250.0),  // Center vertically
                    ..default()
                },
                padding: UiRect::all(Val::Px(20.0)),
                flex_direction: FlexDirection::Column,
                display: Display::None, // Initially hidden
                ..default()
            },
            background_color: Color::rgba(0.0, 0.1, 0.3, 0.95).into(),
            ..default()
        },
        LeaderboardPanel,
        Name::new("Leaderboard Panel"),
    )).with_children(|panel| {
        panel.spawn(TextBundle::from_section(
            "LEADERBOARD",
            TextStyle {
                font: Default::default(), // Use Bevy's default font
                font_size: 24.0,
                color: Color::rgb(1.0, 1.0, 0.2),
            },
        ));
    });
}

// ============================================================================
// KNIRVANA UI UPDATE SYSTEMS
// ============================================================================

/// System to update the NRN display
pub fn update_nrn_display(
    player_resources: Res<PlayerResources>,
    mut query: Query<&mut Text, With<NRNDisplay>>,
) {
    if player_resources.is_changed() {
        for mut text in query.iter_mut() {
            text.sections[0].value = format!("NRN: {:.1}", player_resources.nrn_balance);
        }
    }
}

/// System to update game stats display
pub fn update_game_stats(
    player_resources: Res<PlayerResources>,
    mut query: Query<&mut Text, (Without<NRNDisplay>, With<Text>)>,
) {
    if player_resources.is_changed() {
        for mut text in query.iter_mut() {
            if text.sections[0].value.contains("Errors Resolved") {
                text.sections[0].value = format!(
                    "Errors Resolved: {} | Skills Learned: {}",
                    player_resources.errors_resolved,
                    player_resources.skills_learned
                );
            }
        }
    }
}

/// System to update agent panel with current agent information
pub fn update_agent_panel(
    agent_query: Query<&AIAgent>,
    game_state: Res<KnirvanaGameState>,
    mut panel_query: Query<&Children, With<AgentPanel>>,
    mut text_query: Query<&mut Text>,
    mut commands: Commands,
    asset_server: Res<AssetServer>,
) {
    for children in panel_query.iter() {
        // Find the agent list container
        for &child in children.iter() {
            if let Ok(agent_list_children) = text_query.get(child) {
                // Update agent information
                let mut agent_info = String::new();
                for agent in agent_query.iter() {
                    let status_color = match agent.status {
                        AgentStatus::Idle => "🟢",
                        AgentStatus::Working => "🔵",
                        AgentStatus::Moving => "🟡",
                        AgentStatus::Resting => "🟠",
                        _ => "⚪",
                    };

                    agent_info.push_str(&format!(
                        "{} {} ({:?})\nEnergy: {:.0}/{:.0}\nEfficiency: {:.1}\n\n",
                        status_color,
                        agent.id,
                        agent.specialization,
                        agent.energy,
                        agent.max_energy,
                        agent.efficiency
                    ));
                }

                // Update the text if it exists
                if let Ok(mut text) = text_query.get_mut(child) {
                    if text.sections.len() > 0 && text.sections[0].value != agent_info {
                        text.sections[0].value = agent_info;
                    }
                }
            }
        }
    }
}

/// System to update ErrorNode information panel
pub fn update_error_node_info(
    error_node_query: Query<&ErrorNode>,
    agent_query: Query<&AIAgent>,
    game_state: Res<KnirvanaGameState>,
    mut info_panel_query: Query<&Children, With<ErrorNodeInfo>>,
    mut text_query: Query<&mut Text>,
) {
    for children in info_panel_query.iter() {
        for &child in children.iter() {
            if let Ok(mut text) = text_query.get_mut(child) {
                if text.sections.len() > 0 {
                    if let Some(selected_entity) = game_state.selected_error_node {
                        if let Ok(error_node) = error_node_query.get(selected_entity) {
                            // Find assigned agent if any
                            let mut agent_info = "No agent assigned".to_string();
                            if let Some(agent_id) = &error_node.solver_agent_id {
                                for agent in agent_query.iter() {
                                    if agent.id == *agent_id {
                                        agent_info = format!("Agent: {} ({})", agent.id,
                                            match agent.status {
                                                AgentStatus::Idle => "Idle",
                                                AgentStatus::Working => "Working",
                                                AgentStatus::Moving => "Moving",
                                                AgentStatus::Resting => "Resting",
                                                AgentStatus::Upgrading => "Upgrading",
                                                AgentStatus::Thinking => "Thinking",
                                            });
                                        break;
                                    }
                                }
                            }

                            let info = format!(
                                "ERROR NODE DETAILS\n\n\
                                ID: {}\n\
                                Type: {:?}\n\
                                Difficulty: {:.1}%\n\
                                Bounty: {:.1} NRN\n\
                                Progress: {:.1}%\n\
                                Status: {}\n\
                                {}\n\n\
                                Description:\n{}\n\n\
                                Requirements:\n{}",
                                error_node.id,
                                error_node.node_type,
                                error_node.difficulty * 100.0,
                                error_node.bounty,
                                error_node.progress * 100.0,
                                if error_node.is_being_solved { "Being Solved" } else { "Available" },
                                agent_info,
                                error_node.description,
                                "Click SPACE to deploy agent"
                            );

                            text.sections[0].value = info;
                        } else {
                            text.sections[0].value = "ERROR: Selected node not found".to_string();
                        }
                    } else {
                        // No node selected - show instructions
                        text.sections[0].value = "Click on an error node to see details here.\n\nControls:\n- Left Click: Select node/agent\n- SPACE: Deploy agent\n- TAB: Toggle leaderboard\n- A: Toggle agent panel\n- I: Toggle info panel".to_string();
                    }
                }
            }
        }
    }
}

/// System to update agent thought process display
pub fn update_thought_display(
    agent_query: Query<&AIAgent>,
    game_state: Res<KnirvanaGameState>,
    mut thought_query: Query<&Children, With<ThoughtDisplay>>,
    mut text_query: Query<&mut Text>,
) {
    if let Some(selected_entity) = game_state.selected_agent {
        if let Ok(agent) = agent_query.get(selected_entity) {
            for children in thought_query.iter() {
                for &child in children.iter() {
                    if let Ok(mut text) = text_query.get_mut(child) {
                        if text.sections.len() > 0 && text.sections[0].value.contains("AGENT THOUGHTS") {
                            continue; // Skip the title
                        }

                        let thoughts = agent.thought_process.join("\n");
                        if text.sections.len() > 0 {
                            text.sections[0].value = thoughts;
                        }
                    }
                }
            }
        }
    }
}

/// System to handle UI panel visibility based on state
pub fn handle_ui_visibility(
    ui_state: Res<UIState>,
    mut agent_panel_query: Query<&mut Style, (With<AgentPanel>, Without<ErrorNodeInfo>, Without<LeaderboardPanel>)>,
    mut info_panel_query: Query<&mut Style, (With<ErrorNodeInfo>, Without<AgentPanel>, Without<LeaderboardPanel>)>,
    mut leaderboard_query: Query<&mut Style, (With<LeaderboardPanel>, Without<AgentPanel>, Without<ErrorNodeInfo>)>,
) {
    // Update agent panel visibility
    for mut style in agent_panel_query.iter_mut() {
        style.display = if ui_state.show_agent_panel {
            Display::Flex
        } else {
            Display::None
        };
    }

    // Update info panel visibility
    for mut style in info_panel_query.iter_mut() {
        style.display = if ui_state.show_node_info {
            Display::Flex
        } else {
            Display::None
        };
    }

    // Update leaderboard visibility
    for mut style in leaderboard_query.iter_mut() {
        style.display = if ui_state.show_leaderboard {
            Display::Flex
        } else {
            Display::None
        };
    }
}

/// System to display notifications
pub fn update_notifications(
    mut ui_state: ResMut<UIState>,
    mut notification_query: Query<&Children, With<NotificationPanel>>,
    mut text_query: Query<&mut Text>,
    mut commands: Commands,
    asset_server: Res<AssetServer>,
    time: Res<Time>,
) {
    // Remove expired notifications
    ui_state.notification_queue.retain(|notification| {
        time.elapsed_seconds_f64() - notification.timestamp < notification.duration as f64
    });

    // Update notification display
    for children in notification_query.iter() {
        // Clear existing notifications
        for &child in children.iter() {
            commands.entity(child).despawn_recursive();
        }

        // Add current notifications
        for (i, notification) in ui_state.notification_queue.iter().enumerate() {
            let color = match notification.notification_type {
                NotificationType::Success => Color::rgb(0.0, 1.0, 0.0),
                NotificationType::Warning => Color::rgb(1.0, 1.0, 0.0),
                NotificationType::Error => Color::rgb(1.0, 0.0, 0.0),
                NotificationType::Info => Color::rgb(0.0, 0.8, 1.0),
                NotificationType::Achievement => Color::rgb(1.0, 0.8, 0.0),
            };

            // Spawn notification text
            commands.spawn((
                TextBundle::from_section(
                    &notification.message,
                    TextStyle {
                        font: Default::default(), // Use Bevy's default font
                        font_size: 16.0,
                        color,
                    },
                ).with_style(Style {
                    margin: UiRect::bottom(Val::Px(5.0)),
                    ..default()
                }),
                Name::new(format!("Notification_{}", i)),
            ));
        }
    }
}

/// Helper function to add a notification
pub fn add_notification(
    ui_state: &mut ResMut<UIState>,
    message: String,
    notification_type: NotificationType,
    duration: f32,
    time: &Res<Time>,
) {
    ui_state.notification_queue.push(Notification {
        message,
        notification_type,
        timestamp: time.elapsed_seconds_f64(),
        duration,
    });
}
