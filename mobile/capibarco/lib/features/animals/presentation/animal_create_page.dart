import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:image_picker/image_picker.dart';

import '../../../app/localization/app_localizations.dart';
import '../../../shared/presentation/page_shell.dart';
import '../../../shared/presentation/section_header.dart';
import '../../../shared/presentation/soft_card.dart';
import 'animal_create_controller.dart';

class AnimalCreatePage extends ConsumerStatefulWidget {
  const AnimalCreatePage({super.key});

  @override
  ConsumerState<AnimalCreatePage> createState() => _AnimalCreatePageState();
}

class _AnimalCreatePageState extends ConsumerState<AnimalCreatePage> {
  final _formKey = GlobalKey<FormState>();
  final _nameController = TextEditingController();
  final _breedController = TextEditingController();
  final _ageController = TextEditingController(text: '12');
  final _descriptionController = TextEditingController();
  final _traitsController = TextEditingController();
  final _imagePicker = ImagePicker();

  String _species = 'SPECIES_DOG';
  String _sex = 'ANIMAL_SEX_FEMALE';
  String _size = 'ANIMAL_SIZE_MEDIUM';
  bool _vaccinated = true;
  bool _sterilized = false;
  XFile? _selectedPhoto;
  Uint8List? _selectedPhotoBytes;

  @override
  void dispose() {
    _nameController.dispose();
    _breedController.dispose();
    _ageController.dispose();
    _descriptionController.dispose();
    _traitsController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final state = ref.watch(animalCreateControllerProvider);

    return Scaffold(
      appBar: AppBar(title: Text(l10n.publishPet)),
      body: PageShell(
        child: ListView(
          children: <Widget>[
            SectionHeader(
              title: l10n.publishPet,
              subtitle: 'Create a public animal card for your profile.',
            ),
            const SizedBox(height: 16),
            SoftCard(
              child: Form(
                key: _formKey,
                child: Column(
                  children: <Widget>[
                    TextFormField(
                      controller: _nameController,
                      decoration: InputDecoration(labelText: l10n.petName),
                      validator: (value) =>
                          value == null || value.trim().isEmpty
                          ? 'Required'
                          : null,
                    ),
                    const SizedBox(height: 12),
                    TextFormField(
                      controller: _breedController,
                      decoration: InputDecoration(labelText: l10n.breed),
                    ),
                    const SizedBox(height: 12),
                    DropdownButtonFormField<String>(
                      initialValue: _species,
                      decoration: InputDecoration(labelText: l10n.species),
                      items: const <DropdownMenuItem<String>>[
                        DropdownMenuItem(
                          value: 'SPECIES_DOG',
                          child: Text('Dog'),
                        ),
                        DropdownMenuItem(
                          value: 'SPECIES_CAT',
                          child: Text('Cat'),
                        ),
                        DropdownMenuItem(
                          value: 'SPECIES_RABBIT',
                          child: Text('Rabbit'),
                        ),
                        DropdownMenuItem(
                          value: 'SPECIES_OTHER',
                          child: Text('Other'),
                        ),
                      ],
                      onChanged: (value) => setState(() => _species = value!),
                    ),
                    const SizedBox(height: 12),
                    DropdownButtonFormField<String>(
                      initialValue: _sex,
                      decoration: InputDecoration(labelText: l10n.sex),
                      items: const <DropdownMenuItem<String>>[
                        DropdownMenuItem(
                          value: 'ANIMAL_SEX_FEMALE',
                          child: Text('Female'),
                        ),
                        DropdownMenuItem(
                          value: 'ANIMAL_SEX_MALE',
                          child: Text('Male'),
                        ),
                        DropdownMenuItem(
                          value: 'ANIMAL_SEX_UNKNOWN',
                          child: Text('Unknown'),
                        ),
                      ],
                      onChanged: (value) => setState(() => _sex = value!),
                    ),
                    const SizedBox(height: 12),
                    DropdownButtonFormField<String>(
                      initialValue: _size,
                      decoration: InputDecoration(labelText: l10n.size),
                      items: const <DropdownMenuItem<String>>[
                        DropdownMenuItem(
                          value: 'ANIMAL_SIZE_SMALL',
                          child: Text('Small'),
                        ),
                        DropdownMenuItem(
                          value: 'ANIMAL_SIZE_MEDIUM',
                          child: Text('Medium'),
                        ),
                        DropdownMenuItem(
                          value: 'ANIMAL_SIZE_LARGE',
                          child: Text('Large'),
                        ),
                        DropdownMenuItem(
                          value: 'ANIMAL_SIZE_EXTRA_LARGE',
                          child: Text('Extra large'),
                        ),
                      ],
                      onChanged: (value) => setState(() => _size = value!),
                    ),
                    const SizedBox(height: 12),
                    TextFormField(
                      controller: _ageController,
                      keyboardType: TextInputType.number,
                      decoration: InputDecoration(labelText: l10n.ageMonths),
                    ),
                    const SizedBox(height: 12),
                    TextFormField(
                      controller: _descriptionController,
                      minLines: 3,
                      maxLines: 5,
                      decoration: InputDecoration(labelText: l10n.bio),
                    ),
                    const SizedBox(height: 12),
                    TextFormField(
                      controller: _traitsController,
                      decoration: const InputDecoration(
                        labelText: 'Traits',
                        hintText: 'friendly, calm, playful',
                      ),
                    ),
                    const SizedBox(height: 12),
                    Align(
                      alignment: Alignment.centerLeft,
                      child: Text(
                        l10n.petPhoto,
                        style: Theme.of(context).textTheme.titleMedium
                            ?.copyWith(fontWeight: FontWeight.w800),
                      ),
                    ),
                    const SizedBox(height: 10),
                    if (_selectedPhoto != null &&
                        _selectedPhotoBytes != null) ...<Widget>[
                      ClipRRect(
                        borderRadius: BorderRadius.circular(24),
                        child: SizedBox(
                          height: 180,
                          width: double.infinity,
                          child: Image.memory(
                            _selectedPhotoBytes!,
                            fit: BoxFit.cover,
                          ),
                        ),
                      ),
                      const SizedBox(height: 12),
                    ],
                    Row(
                      children: <Widget>[
                        Expanded(
                          child: OutlinedButton.icon(
                            onPressed: state.isSubmitting ? null : _pickPhoto,
                            icon: const Icon(Icons.add_a_photo_rounded),
                            label: Text(
                              _selectedPhoto == null
                                  ? l10n.addPhoto
                                  : l10n.changePhoto,
                            ),
                          ),
                        ),
                        if (_selectedPhoto != null) ...<Widget>[
                          const SizedBox(width: 12),
                          IconButton.filledTonal(
                            onPressed: state.isSubmitting
                                ? null
                                : () => setState(() {
                                    _selectedPhoto = null;
                                    _selectedPhotoBytes = null;
                                  }),
                            icon: const Icon(Icons.delete_outline_rounded),
                          ),
                        ],
                      ],
                    ),
                    const SizedBox(height: 12),
                    SwitchListTile.adaptive(
                      contentPadding: EdgeInsets.zero,
                      title: Text(l10n.vaccinated),
                      value: _vaccinated,
                      onChanged: (value) => setState(() => _vaccinated = value),
                    ),
                    SwitchListTile.adaptive(
                      contentPadding: EdgeInsets.zero,
                      title: Text(l10n.sterilized),
                      value: _sterilized,
                      onChanged: (value) => setState(() => _sterilized = value),
                    ),
                    if (state.errorMessage != null) ...<Widget>[
                      const SizedBox(height: 12),
                      Text(
                        state.errorMessage!,
                        style: TextStyle(
                          color: Theme.of(context).colorScheme.error,
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                    ],
                    const SizedBox(height: 16),
                    FilledButton(
                      onPressed: state.isSubmitting
                          ? null
                          : () async {
                              if (!_formKey.currentState!.validate()) {
                                return;
                              }
                              final success = await ref
                                  .read(animalCreateControllerProvider.notifier)
                                  .createAnimal(
                                    name: _nameController.text.trim(),
                                    species: _species,
                                    breed: _breedController.text.trim(),
                                    sex: _sex,
                                    size: _size,
                                    ageMonths:
                                        int.tryParse(
                                          _ageController.text.trim(),
                                        ) ??
                                        0,
                                    description: _descriptionController.text
                                        .trim(),
                                    traits: _traitsController.text
                                        .split(',')
                                        .map((item) => item.trim())
                                        .where((item) => item.isNotEmpty)
                                        .toList(),
                                    vaccinated: _vaccinated,
                                    sterilized: _sterilized,
                                    photo: _selectedPhoto,
                                    photoBytes: _selectedPhotoBytes,
                                  );
                              if (success && context.mounted) {
                                ScaffoldMessenger.of(context).showSnackBar(
                                  SnackBar(content: Text(l10n.animalCreated)),
                                );
                                context.pop();
                              }
                            },
                      child: state.isSubmitting
                          ? const SizedBox(
                              width: 20,
                              height: 20,
                              child: CircularProgressIndicator(
                                strokeWidth: 2.4,
                              ),
                            )
                          : Text(l10n.publishPet),
                    ),
                  ],
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _pickPhoto() async {
    final photo = await _imagePicker.pickImage(
      source: ImageSource.gallery,
      imageQuality: 88,
    );
    if (photo == null || !mounted) {
      return;
    }
    final photoBytes = await photo.readAsBytes();
    if (!mounted) {
      return;
    }
    setState(() {
      _selectedPhoto = photo;
      _selectedPhotoBytes = photoBytes;
    });
  }
}
