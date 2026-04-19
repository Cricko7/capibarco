import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';

import '../../../app/localization/app_localizations.dart';
import '../../../shared/presentation/page_shell.dart';
import '../../../shared/presentation/section_header.dart';
import '../../../shared/presentation/soft_card.dart';
import '../../../shared/presentation/stale_banner.dart';
import '../../../shared/presentation/status_view.dart';
import '../../auth/presentation/auth_controller.dart';
import '../../chat/presentation/chat_repository_provider.dart';
import '../domain/entities/notification_item.dart';
import 'notifications_controller.dart';

class NotificationsPage extends ConsumerStatefulWidget {
  const NotificationsPage({super.key});

  @override
  ConsumerState<NotificationsPage> createState() => _NotificationsPageState();
}

class _NotificationsPageState extends ConsumerState<NotificationsPage> {
  Timer? _refreshTimer;

  @override
  void initState() {
    super.initState();
    _refreshTimer = Timer.periodic(const Duration(seconds: 6), (_) {
      if (!mounted) {
        return;
      }
      ref.read(notificationsControllerProvider.notifier).refreshRealtime();
    });
  }

  @override
  void dispose() {
    _refreshTimer?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final state = ref.watch(notificationsControllerProvider);
    ref.listen<String>(
      authControllerProvider.select((auth) => auth.session?.user.id ?? ''),
      (previous, next) {
        if (next.isEmpty || previous == next) {
          return;
        }
        Future<void>.microtask(() {
          if (!mounted) {
            return;
          }
          ref.read(notificationsControllerProvider.notifier).load();
        });
      },
    );
    if (state.profileId.isNotEmpty && !state.isLoading && !state.hasLoaded) {
      Future<void>.microtask(() {
        if (!mounted) {
          return;
        }
        ref.read(notificationsControllerProvider.notifier).load();
      });
    }
    final dateFormat = DateFormat.yMMMd(
      Localizations.localeOf(context).languageCode,
    ).add_Hm();

    return Scaffold(
      body: PageShell(
        child: RefreshIndicator(
          onRefresh: () =>
              ref.read(notificationsControllerProvider.notifier).load(),
          child: ListView(
            children: <Widget>[
              SectionHeader(
                title: l10n.notifications,
                subtitle: state.items.isEmpty
                    ? 'Adoption responses and chat updates arrive here.'
                    : '${state.items.length} update${state.items.length == 1 ? '' : 's'} ready.',
              ),
              if (state.isStale) ...<Widget>[
                const SizedBox(height: 16),
                StaleBanner(message: l10n.staleData),
              ],
              const SizedBox(height: 16),
              if (state.isLoading && state.items.isEmpty)
                StatusView.loading(message: l10n.loading)
              else if (state.errorMessage != null && state.items.isEmpty)
                StatusView.message(
                  message: state.errorMessage!,
                  icon: Icons.error_outline_rounded,
                  action: FilledButton(
                    onPressed: () => ref
                        .read(notificationsControllerProvider.notifier)
                        .load(),
                    child: Text(l10n.retry),
                  ),
                )
              else if (state.items.isEmpty)
                StatusView.message(
                  message: l10n.emptyNotifications,
                  icon: Icons.notifications_none_rounded,
                )
              else
                ...state.items.map(
                  (item) => Padding(
                    padding: const EdgeInsets.only(bottom: 16),
                    child: SoftCard(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: <Widget>[
                          Row(
                            children: <Widget>[
                              Expanded(
                                child: Text(
                                  item.title,
                                  style: Theme.of(context).textTheme.titleLarge
                                      ?.copyWith(fontWeight: FontWeight.w800),
                                ),
                              ),
                              if (item.readAt == null)
                                FilledButton.tonal(
                                  style: _notificationActionButtonStyle,
                                  onPressed: () => ref
                                      .read(
                                        notificationsControllerProvider
                                            .notifier,
                                      )
                                      .markAsRead(item),
                                  child: const Text('Mark read'),
                                ),
                            ],
                          ),
                          const SizedBox(height: 8),
                          Text(item.body),
                          if (_canStartChat(item)) ...[
                            const SizedBox(height: 12),
                            FilledButton.icon(
                              style: _notificationActionButtonStyle,
                              onPressed: () => _openConversation(item),
                              icon: const Icon(Icons.chat_bubble_rounded),
                              label: Text(l10n.startChat),
                            ),
                          ],
                          const SizedBox(height: 12),
                          Text(
                            '${item.type} · ${dateFormat.format(item.createdAt.toLocal())}',
                            style: Theme.of(context).textTheme.bodySmall,
                          ),
                        ],
                      ),
                    ),
                  ),
                ),
            ],
          ),
        ),
      ),
    );
  }

  Future<void> _openConversation(NotificationItemEntity item) async {
    final l10n = AppLocalizations.of(context);
    if (item.readAt == null) {
      try {
        await ref
            .read(notificationsControllerProvider.notifier)
            .markAsRead(item);
      } catch (_) {
        // Opening the chat is more important than read-state sync here.
      }
    }
    if (!mounted) {
      return;
    }
    final existingConversationId = item.data['conversation_id'] ?? '';
    final targetProfileId = item.data['adopter_profile_id'] ?? '';
    final currentProfileId =
        ref.read(authControllerProvider).session?.user.id ?? '';
    final counterpartProfileId =
        targetProfileId.isNotEmpty && targetProfileId != currentProfileId
        ? targetProfileId
        : '';
    if (existingConversationId.isNotEmpty) {
      final destination = Uri(
        path: '/chat/$existingConversationId',
        queryParameters: <String, String>{
          'return_to': '/notifications',
          if (counterpartProfileId.isNotEmpty)
            'profile_id': counterpartProfileId,
        },
      ).toString();
      context.go(destination);
      return;
    }

    if (targetProfileId.isEmpty) {
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text(l10n.chatUnavailable)));
      return;
    }

    try {
      final conversation = await ref
          .read(chatRepositoryProvider)
          .createConversation(
            targetProfileId: targetProfileId,
            animalId: item.data['animal_id'] ?? '',
            matchId: item.data['match_id'] ?? '',
            idempotencyKey: 'notification-${item.id}-chat',
          );
      if (!mounted) {
        return;
      }
      final destination = Uri(
        path: '/chat/${conversation.id}',
        queryParameters: <String, String>{
          'return_to': '/notifications',
          'profile_id': targetProfileId,
        },
      ).toString();
      context.go(destination);
    } catch (_) {
      if (!mounted) {
        return;
      }
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text(l10n.chatUnavailable)));
    }
  }

  bool _canStartChat(NotificationItemEntity item) {
    final conversationId = item.data['conversation_id'] ?? '';
    final adopterProfileId = item.data['adopter_profile_id'] ?? '';
    return conversationId.isNotEmpty || adopterProfileId.isNotEmpty;
  }
}

const _notificationActionButtonStyle = ButtonStyle(
  minimumSize: WidgetStatePropertyAll<Size>(Size(0, 46)),
  tapTargetSize: MaterialTapTargetSize.shrinkWrap,
);
