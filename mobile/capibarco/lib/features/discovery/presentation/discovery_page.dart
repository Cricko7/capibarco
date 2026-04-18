import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../app/localization/app_localizations.dart';
import '../../../shared/presentation/page_shell.dart';
import '../../../shared/presentation/section_header.dart';
import '../../../shared/presentation/soft_card.dart';
import '../../../shared/presentation/stale_banner.dart';
import '../../../shared/presentation/status_view.dart';
import 'discovery_controller.dart';

class DiscoveryPage extends ConsumerStatefulWidget {
  const DiscoveryPage({super.key});

  @override
  ConsumerState<DiscoveryPage> createState() => _DiscoveryPageState();
}

class _DiscoveryPageState extends ConsumerState<DiscoveryPage> {
  final _queryController = TextEditingController();
  final _cityController = TextEditingController();

  @override
  void initState() {
    super.initState();
    Future<void>.microtask(
      () => ref
          .read(discoveryControllerProvider.notifier)
          .search(query: '', city: ''),
    );
  }

  @override
  void dispose() {
    _queryController.dispose();
    _cityController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final state = ref.watch(discoveryControllerProvider);

    return Scaffold(
      body: PageShell(
        child: ListView(
          children: <Widget>[
            SectionHeader(
              title: l10n.discover,
              subtitle:
                  'Search across public profiles exposed by user-service.',
            ),
            const SizedBox(height: 16),
            SoftCard(
              child: Column(
                children: <Widget>[
                  TextField(
                    controller: _queryController,
                    decoration: InputDecoration(
                      labelText: l10n.searchProfiles,
                      hintText: l10n.searchHint,
                      prefixIcon: const Icon(Icons.search_rounded),
                    ),
                  ),
                  const SizedBox(height: 12),
                  TextField(
                    controller: _cityController,
                    decoration: InputDecoration(
                      labelText: l10n.city,
                      prefixIcon: const Icon(Icons.location_on_outlined),
                    ),
                  ),
                  const SizedBox(height: 16),
                  FilledButton(
                    onPressed: state.isLoading
                        ? null
                        : () => ref
                              .read(discoveryControllerProvider.notifier)
                              .search(
                                query: _queryController.text.trim(),
                                city: _cityController.text.trim(),
                              ),
                    child: state.isLoading
                        ? const SizedBox(
                            width: 20,
                            height: 20,
                            child: CircularProgressIndicator(strokeWidth: 2.4),
                          )
                        : Text(l10n.searchProfiles),
                  ),
                ],
              ),
            ),
            if (state.isStale) ...<Widget>[
              const SizedBox(height: 16),
              StaleBanner(message: l10n.staleData),
            ],
            const SizedBox(height: 16),
            if (state.errorMessage != null && state.items.isEmpty)
              StatusView.message(
                message: state.errorMessage!,
                icon: Icons.error_outline_rounded,
              )
            else if (!state.isLoading && state.items.isEmpty)
              StatusView.message(
                message: l10n.emptyProfiles,
                icon: Icons.travel_explore_rounded,
              )
            else
              ...state.items.map(
                (profile) => Padding(
                  padding: const EdgeInsets.only(bottom: 16),
                  child: SoftCard(
                    padding: EdgeInsets.zero,
                    child: Material(
                      color: Colors.transparent,
                      child: InkWell(
                        borderRadius: BorderRadius.circular(28),
                        onTap: () => context.push('/profiles/${profile.id}'),
                        child: Padding(
                          padding: const EdgeInsets.all(20),
                          child: Row(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: <Widget>[
                              CircleAvatar(
                                radius: 30,
                                backgroundColor: Theme.of(
                                  context,
                                ).colorScheme.primaryContainer,
                                backgroundImage: profile.avatarUrl.isNotEmpty
                                    ? NetworkImage(profile.avatarUrl)
                                    : null,
                                child: profile.avatarUrl.isEmpty
                                    ? const Icon(Icons.person_rounded)
                                    : null,
                              ),
                              const SizedBox(width: 16),
                              Expanded(
                                child: Column(
                                  crossAxisAlignment: CrossAxisAlignment.start,
                                  children: <Widget>[
                                    Text(
                                      profile.displayName,
                                      style: Theme.of(context)
                                          .textTheme
                                          .titleLarge
                                          ?.copyWith(
                                            fontWeight: FontWeight.w800,
                                          ),
                                    ),
                                    const SizedBox(height: 4),
                                    Text(
                                      '${profile.typeLabel}${profile.city.isNotEmpty ? ' В· ${profile.city}' : ''}',
                                      style: Theme.of(
                                        context,
                                      ).textTheme.bodyMedium,
                                    ),
                                    const SizedBox(height: 8),
                                    Text(
                                      profile.bio.isEmpty
                                          ? 'No profile bio yet.'
                                          : profile.bio,
                                    ),
                                    const SizedBox(height: 10),
                                    Row(
                                      children: <Widget>[
                                        const Icon(
                                          Icons.star_rounded,
                                          size: 18,
                                        ),
                                        const SizedBox(width: 4),
                                        Text(
                                          '${profile.averageRating.toStringAsFixed(1)} (${profile.reviewsCount})',
                                        ),
                                        const Spacer(),
                                        Text(
                                          l10n.openProfile,
                                          style: TextStyle(
                                            color: Theme.of(
                                              context,
                                            ).colorScheme.primary,
                                            fontWeight: FontWeight.w700,
                                          ),
                                        ),
                                      ],
                                    ),
                                  ],
                                ),
                              ),
                            ],
                          ),
                        ),
                      ),
                    ),
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }
}
