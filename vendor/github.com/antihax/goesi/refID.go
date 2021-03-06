package goesi

// GetJournalRefID looks up the Journal reference name and returns the internal ID
// WARNING: These are subject to change per CCP.
func GetJournalRefID(referenceName string) int {
	return JournalRefID[referenceName]
}

// JournalRefID Maps journal strings to CCP internal refID. CCP has stated these are subject to change.
// Please submit pull requests for new IDs
var JournalRefID = map[string]int{
	"player_trading":                                  1,
	"market_transaction":                              2,
	"gm_cash_transfer":                                3,
	"mission_reward":                                  7,
	"clone_activation":                                8,
	"inheritance":                                     9,
	"player_donation":                                 10,
	"corporation_payment":                             11,
	"docking_fee":                                     12,
	"office_rental_fee":                               13,
	"factory_slot_rental_fee":                         14,
	"repair_bill":                                     15,
	"bounty":                                          16,
	"bounty_prize":                                    17,
	"insurance":                                       19,
	"mission_expiration":                              20,
	"mission_completion":                              21,
	"shares":                                          22,
	"courier_mission_escrow":                          23,
	"mission_cost":                                    24,
	"agent_miscellaneous":                             25,
	"lp_store":                                        26,
	"agent_location_services":                         27,
	"agent_donation":                                  28,
	"agent_security_services":                         29,
	"agent_mission_collateral_paid":                   30,
	"agent_mission_collateral_refunded":               31,
	"agents_preward":                                  32,
	"agent_mission_reward":                            33,
	"agent_mission_time_bonus_reward":                 34,
	"cspa":                                            35,
	"cspaofflinerefund":                               36,
	"corporation_account_withdrawal":                  37,
	"corporation_dividend_payment":                    38,
	"corporation_registration_fee":                    39,
	"corporation_logo_change_cost":                    40,
	"release_of_impounded_property":                   41,
	"market_escrow":                                   42,
	"agent_services_rendered":                         43,
	"market_fine_paid":                                44,
	"corporation_liquidation":                         45,
	"brokers_fee":                                     46,
	"corporation_bulk_payment":                        47,
	"alliance_registration_fee":                       48,
	"war_fee":                                         49,
	"alliance_maintainance_fee":                       50,
	"contraband_fine":                                 51,
	"clone_transfer":                                  52,
	"acceleration_gate_fee":                           53,
	"transaction_tax":                                 54,
	"jump_clone_installation_fee":                     55,
	"manufacturing":                                   56,
	"researching_technology":                          57,
	"researching_time_productivity":                   58,
	"researching_material_productivity":               59,
	"copying":                                         60,
	"reverse_engineering":                             62,
	"contract_auction_bid":                            63,
	"contract_auction_bid_refund":                     64,
	"contract_collateral":                             65,
	"contract_reward_refund":                          66,
	"contract_auction_sold":                           67,
	"contract_reward":                                 68,
	"contract_collateral_refund":                      69,
	"contract_collateral_payout":                      70,
	"contract_price":                                  71,
	"contract_brokers_fee":                            72,
	"contract_sales_tax":                              73,
	"contract_deposit":                                74,
	"contract_deposit_sales_tax":                      75,
	"contract_auction_bid_corp":                       77,
	"contract_collateral_deposited_corp":              78,
	"contract_price_payment_corp":                     79,
	"contract_brokers_fee_corp":                       80,
	"contract_deposit_corp":                           81,
	"contract_deposit_refund":                         82,
	"contract_reward_deposited":                       83,
	"contract_reward_deposited_corp":                  84,
	"bounty_prizes":                                   85,
	"advertisement_listing_fee":                       86,
	"medal_creation":                                  87,
	"medal_issued":                                    88,
	"dna_modification_fee":                            90,
	"sovereignity_bill":                               91,
	"bounty_prize_corporation_tax":                    92,
	"agent_mission_reward_corporation_tax":            93,
	"agent_mission_time_bonus_reward_corporation_tax": 94,
	"upkeep_adjustment_fee":                           95,
	"planetary_import_tax":                            96,
	"planetary_export_tax":                            97,
	"planetary_construction":                          98,
	"corporate_reward_payout":                         99,
	"bounty_surcharge":                                101,
	"contract_reversal":                               102,
	"corporate_reward_tax":                            103,
	"store_purchase":                                  106,
	"store_purchase_refund":                           107,
	"datacore_fee":                                    112,
	"war_fee_surrender":                               113,
	"war_ally_contract":                               114,
	"bounty_reimbursement":                            115,
	"kill_right_fee":                                  116,
	"security_processing_fee":                         117,
	"industry_job_tax":                                120,
	"infrastructure_hub_maintenance":                  122,
	"asset_safety_recovery_tax":                       123,
	"opportunity_reward":                              124,
	"project_discovery_reward":                        125,
	"project_discovery_tax":                           126,
	"reprocessing_tax":                                127,
	"jump_clone_activation_fee":                       128,
	"operation_bonus":                                 129,
	"resource_wars_reward":                            131,
	"duel_wager_escrow":                               132,
	"duel_wager_payment":                              133,
	"duel_wager_refund":                               134,
	"reaction":                                        135,
	"undefined":                                       0,
	"atm_withdraw":                                    4,
	"atm_deposit":                                     5,
	"backward_compatible":                             6,
	"agents_temporary":                                18,
	"duplicating":                                     61,
	"secure_eve_time_code_exchange":                   76,
	"contract_auction_bid_(corp)":                     77,
	"contract_collateral_deposited_(corp)":            78,
	"contract_price_payment_(corp)":                   79,
	"contract_brokers_fee_(corp)":                     80,
	"contract_deposit_(corp)":                         81,
	"contract_reward_deposited_(corp)":                84,
	"betting":                                         89,
	"plex_sold_for_aurum":                             108,
	"lottery_give_away":                               109,
	"aurum_token_exchanged_for_aur":                   111,
	"escrow_for_industry_team_auction":                118,
	"reimbursement_of_escrow":                         119,
	"modify_isk":                                      10001,
	"primary_marketplace_purchase":                    10002,
	"battle_reward":                                   10003,
	"new_character_starting_funds":                    10004,
	"corporation_account_deposit":                     10006,
	"battle_wp_win_reward":                            10007,
	"battle_wp_loss_reward":                           10008,
	"battle_win_reward":                               10009,
	"battle_loss_reward":                              10010,
	"reset_isk_for_character_reset":                   10011,
	"district_contract_deposit":                       10012,
	"district_contract_deposit_refund":                10013,
	"district_contract_collateral":                    10014,
	"district_contract_collateral_refund":             10015,
	"district_contract_reward":                        10016,
	"district_clone_transportation":                   10017,
	"district_clone_transportation_refund":            10018,
	"district_infrastructure":                         10019,
	"district_clone_sales":                            10020,
	"district_clone_purchase":                         10021,
	"biomass_reward":                                  10022,
	"isk_swap_reward":                                 10023,
	"modify_aur":                                      11001,
	"respec_payment":                                  11002,
	"entitlement":                                     11003,
	"reset_reimbursement":                             11004,
	"reset_aur_for_character_reset":                   11005,
	"daily_mission_cp":                                12001,
	"warbarge_cp":                                     12002,
	"donate_cp":                                       12003,
	"use_cp_for_clone_packs":                          12004,
	"use_cp_for_moving_clones":                        12005,
	"use_cp_for_selling_clones":                       12006,
	"use_cp_for_changing_reinforcement":               12007,
	"use_cp_for_changing_surface_infrastructure":      12008,
	"daily_mission_dk":                                13001,
	"planetary_conquest_dk":                           13002,
	"use_dk_for_purchasing_items":                     13003,
	"use_dk_for_rerolling_market":                     13004,
	"selling_clones_dk":                               13005,
}
