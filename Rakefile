namespace :braile do
  desc "group chars by number of dots"
  task :by_dots do
    require 'unicode/name'

    h = {}
    256.times.map{ |i| [0x2800 + i].pack('U') }.each do |c|
      name = Unicode::Name.of(c)
      ndots = 0
      if name =~ /BRAILLE PATTERN DOTS-(\d+)$/
        ndots = $1.to_s.size
      end

      h[ndots] ||= ""
      h[ndots] << c
    end

    h.keys.sort.each do |ndots|
      puts "#{ndots}: #{h[ndots].inspect}"
    end
  end

  desc "group chars by number code"
  task :by_code do
    0x2800.upto(0x28FF).to_a.each_slice(0x20) do |slice|
      s = ""
      slice.each do |code|
        s << [code].pack('U')
      end
      printf "%s // %02x-%02x\n", s.inspect, slice.first, slice.last
    end
  end
end
